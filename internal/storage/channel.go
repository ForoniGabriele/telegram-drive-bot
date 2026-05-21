package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"tg-drive-bot/internal/model"

	tele "gopkg.in/telebot.v4"
)

// 当所有的副本都发送失败时返回的错误
// 用于决定是否回退到direct模式
var ErrAllCopiesFailed = errors.New("all storage copies failed")

// Storage接口, 目前有两种实现:
//
//	channel: 转发到频道
//	direct: 使用bot的file_id
type Storage interface {
	// 将文件存储在所配置的存储频道中
	// 返回保存成功的副本位置（成功时至少包含一个）, 在 direct 模式下，返回空切片
	// 仅在需要频道存储但所有目标位置均失败时返回错误
	ForwardToStorage(bot tele.API, msg *tele.Message) ([]model.CopyLocation, error)

	// 将存储的文件发送给用户，并返回发送成功的消息
	// copies 是该文件副本在已知的存储channel中的位置（在 direct 模式下为空）
	// caption 在两种模式下都会作为消息的 caption 发送 -- channel 模式通过 copyMessage 的 caption 参数覆盖原始 caption,
	// 与 direct 模式表现一致
	// 当 channel 模式存在副本但全部发送失败时，返回 ErrAllCopiesFailed，以便代码决定是否回退到 direct 模式发送
	SendFileToUser(bot tele.API, userChat *tele.Chat, file *model.File, copies []model.CopyLocation, caption string) (*tele.Message, error)

	// 从存储中移除给定的副本
	// 在 direct 模式下，这是一个空操作, 当 Channel 的 delete-copies 标志位被禁用时，此调用也是一个空操作
	// 在 channel 模式下，每个副本消息都会通过 Telegram Bot API 的 deleteMessage 进行删除
	// "消息已不存在(Message already gone)"类的错误会被忽略, 做"尽力而为"的删除
	// 除此之外的任何其他类型失败都会导致操作中止并返回错误
	DeleteFromStorage(bot tele.API, copies []model.CopyLocation) error
}

// healthTracker tracks per-channel failure counts and temporarily disables channels
// that hit the threshold. State is in-memory and resets on process restart.
type healthTracker struct {
	mu             sync.Mutex
	failureCount   map[int64]int
	disabledUntil  map[int64]time.Time
	failureLimit   int
	cooldownPeriod time.Duration
}

func newHealthTracker(failureLimit int, cooldown time.Duration) *healthTracker {
	return &healthTracker{
		failureCount:   make(map[int64]int),
		disabledUntil:  make(map[int64]time.Time),
		failureLimit:   failureLimit,
		cooldownPeriod: cooldown,
	}
}

// isHealthy reports whether the channel is currently eligible for use.
// A disabled channel becomes eligible again after the cooldown elapses.
func (h *healthTracker) isHealthy(chatID int64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	until, disabled := h.disabledUntil[chatID]
	if !disabled {
		return true
	}
	if time.Now().After(until) {
		delete(h.disabledUntil, chatID)
		h.failureCount[chatID] = 0
		return true
	}
	return false
}

// ++失败计数器，并在达到阈值（limit）时禁用该频道
func (h *healthTracker) markFailure(chatID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.failureCount[chatID]++
	if h.failureCount[chatID] >= h.failureLimit {
		h.disabledUntil[chatID] = time.Now().Add(h.cooldownPeriod)
		slog.Warn("storage channel disabled after repeated failures",
			"chat_id", chatID, "cooldown", h.cooldownPeriod, "failures", h.failureCount[chatID])
	}
}

// markFlood 处理 Telegram 429 Too Many Requests:直接按 retry_after 精准冷却,
// 不走 failureLimit 阈值(429 已经明示要等多久,继续打就是火上浇油)
// retryAfter 上加 1 秒 buffer 抵消时钟漂移;<=0 时降级用默认 cooldownPeriod
func (h *healthTracker) markFlood(chatID int64, retryAfter time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	cooldown := retryAfter + time.Second
	if retryAfter <= 0 {
		cooldown = h.cooldownPeriod
	}
	h.disabledUntil[chatID] = time.Now().Add(cooldown)
	// 清零普通失败计数,避免和 429 路径相互串扰
	delete(h.failureCount, chatID)
	slog.Warn("storage channel 429, cooling down by retry_after",
		"chat_id", chatID, "retry_after", retryAfter, "cooldown", cooldown)
}

// markFailureFromError 根据错误类型选择失败处理路径:
// - 若是 telebot 的 FloodError(HTTP 429),按其 RetryAfter 精准冷却
// - 否则走普通的累计失败计数
func (h *healthTracker) markFailureFromError(chatID int64, err error) {
	var fe tele.FloodError
	if errors.As(err, &fe) {
		h.markFlood(chatID, time.Duration(fe.RetryAfter)*time.Second)
		return
	}
	h.markFailure(chatID)
}

// markSuccess clears any accumulated failure state for a channel.
func (h *healthTracker) markSuccess(chatID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.failureCount[chatID] != 0 {
		delete(h.failureCount, chatID)
	}
	delete(h.disabledUntil, chatID)
}

// Channel implements Storage by forwarding files to one or more Telegram channels.
type Channel struct {
	storageChatIDs []int64
	deleteCopies   bool
	health         *healthTracker
}

// 创建一个新的 Channel 实例，并绑定给定的存储频道 ID
// 第一个 ID 被视为主要存储目标，其余均为冗余副本
// deleteCopies 控制 DeleteFromStorage 是否实际删除频道中的消息：
//
//	当为 false 时，删除操作会跳过 Telegram API，仅清理数据库中的记录
//
// failureLimit和 cooldownPeriod用于控制每个频道的熔断机制
func NewChannel(storageChatIDs []int64, deleteCopies bool, failureLimit int, cooldownPeriod time.Duration) *Channel {
	return &Channel{
		storageChatIDs: storageChatIDs,
		deleteCopies:   deleteCopies,
		health:         newHealthTracker(failureLimit, cooldownPeriod),
	}
}

// 将消息复制到每个配置的存储频道中, 会优先尝试状态健康的频道
// 按配置顺序返回保存成功的副本位置列表
// 如果所有目标频道都失败，则返回错误
func (ch *Channel) ForwardToStorage(bot tele.API, msg *tele.Message) ([]model.CopyLocation, error) {
	if len(ch.storageChatIDs) == 0 {
		return nil, fmt.Errorf("no storage channels configured")
	}

	copies := make([]model.CopyLocation, 0, len(ch.storageChatIDs))
	var lastErr error

	for _, chatID := range ch.storageChatIDs {
		if !ch.health.isHealthy(chatID) {
			continue
		}
		loc, err := ch.copyOne(bot, msg, chatID)
		if err != nil {
			lastErr = err
			ch.health.markFailureFromError(chatID, err)
			slog.Warn("forward to storage channel failed", "error", err, "chat_id", chatID)
			continue
		}
		ch.health.markSuccess(chatID)
		copies = append(copies, loc)
	}

	if len(copies) == 0 {
		return nil, fmt.Errorf("forward to all storage channels failed: %w", lastErr)
	}
	return copies, nil
}

// copyOne performs a single Copy call to one storage channel.
func (ch *Channel) copyOne(bot tele.API, msg *tele.Message, chatID int64) (model.CopyLocation, error) {
	storageChat := &tele.Chat{ID: chatID}
	copied, err := bot.Copy(storageChat, msg)
	if err != nil {
		return model.CopyLocation{}, err
	}
	return model.CopyLocation{ChatID: chatID, MsgID: copied.ID}, nil
}

// 按顺序尝试发送每个副本，直到成功为止，并返回发送成功的消息
// 通过 copyMessageWithCaption 强制覆盖 caption,使 channel 模式与 direct 模式表现一致
// 当没有任何副本发送成功或未提供副本列表时，返回 ErrAllCopiesFailed
func (ch *Channel) SendFileToUser(bot tele.API, userChat *tele.Chat, file *model.File, copies []model.CopyLocation, caption string) (*tele.Message, error) {
	if len(copies) == 0 {
		return nil, ErrAllCopiesFailed
	}

	var lastErr error
	for _, loc := range copies {
		if !ch.health.isHealthy(loc.ChatID) {
			continue
		}
		storedMsg := &tele.Message{ID: loc.MsgID, Chat: &tele.Chat{ID: loc.ChatID}}
		sent, err := copyMessageWithCaption(bot, userChat, storedMsg, caption)
		if err != nil {
			lastErr = err
			ch.health.markFailureFromError(loc.ChatID, err)
			slog.Warn("send from storage channel failed", "error", err, "chat_id", loc.ChatID, "file_db_id", file.ID)
			continue
		}
		ch.health.markSuccess(loc.ChatID)

		// Telegram 的 copyMessage 响应只包含 message_id，其 Chat 字段为 nil
		// 在此进行补全，以便调用者可以安全地对返回的消息调用 EditReplyMarkup 或 MessageSig
		if sent.Chat == nil {
			sent.Chat = userChat
		}
		return sent, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no healthy storage channel available")
	}
	return nil, fmt.Errorf("%w: %v", ErrAllCopiesFailed, lastErr)
}

// 通过 deleteMessage 将给定的副本从对应的存储频道中删除
// 当 Channel 的 deleteCopies 标志位为 false 时调用为一个空操作: 清理数据库记录，但会保留频道内的消息副本
// “Message already gone”类的错误将被视为删除成功
// 除此之外任何其他类型的失败都会导致操作中止并返回错误
func (ch *Channel) DeleteFromStorage(bot tele.API, copies []model.CopyLocation) error {
	if !ch.deleteCopies {
		slog.Info("DELETE_CHANNEL_COPIES=false — skipping storage deletion", "copies", len(copies))
		return nil
	}
	for _, loc := range copies {
		storedMsg := &tele.Message{ID: loc.MsgID, Chat: &tele.Chat{ID: loc.ChatID}}
		if err := bot.Delete(storedMsg); err != nil {
			if isMessageGoneErr(err) {
				slog.Info("storage msg already gone", "chat_id", loc.ChatID, "msg_id", loc.MsgID, "error", err)
				continue
			}
			return fmt.Errorf("delete copy (chat=%d msg=%d) failed: %w", loc.ChatID, loc.MsgID, err)
		}
	}
	return nil
}

// 检查错误是否为消息已不存在
// 这种情况在删除过程中默认是可以容忍的
func isMessageGoneErr(err error) bool {
	if errors.Is(err, tele.ErrNotFoundToDelete) {
		return true
	}
	s := err.Error()
	return strings.Contains(s, "message to delete not found") ||
		strings.Contains(s, "MESSAGE_ID_INVALID") ||
		strings.Contains(s, "message not found")
}

// copyMessageWithCaption 通过 Bot API 的 copyMessage 端点复制一条消息,并允许覆盖 caption
// telebot.v4 的 Bot.Copy 没有暴露 caption 参数,这里直接走 Raw 构造请求
// caption 以纯文本形式发送(不传 parse_mode),避免原始 caption 中的特殊字符被误判为 Markdown/HTML
// 截断由调用方在 buildXxxCaption 阶段完成, 这里不再处理长度
//
// Raw 内部已经处理了 API 错误, 当返回 err == nil 时, data 一定包含一个有效的 result 字段
func copyMessageWithCaption(bot tele.API, to *tele.Chat, from *tele.Message, caption string) (*tele.Message, error) {
	params := map[string]string{
		"chat_id":      strconv.FormatInt(to.ID, 10),
		"from_chat_id": strconv.FormatInt(from.Chat.ID, 10),
		"message_id":   strconv.Itoa(from.ID),
		"caption":      caption,
	}

	data, err := bot.Raw("copyMessage", params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Result *tele.Message `json:"result"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode copyMessage response: %w", err)
	}
	if resp.Result == nil {
		return nil, fmt.Errorf("copyMessage returned empty result")
	}
	return resp.Result, nil
}
