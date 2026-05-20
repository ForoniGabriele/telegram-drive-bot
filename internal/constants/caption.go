package constants

// CaptionMaxUTF16 是 Telegram media caption 的长度上限,以 UTF-16 code units 计
// Telegram 服务器对 caption 长度的判断与 JavaScript 的 String.length 相同,
// 因此长度计算需要用 UTF-16 code units 而非 Go 的 rune 数或字节数
// 见: https://core.telegram.org/bots/api#sendphoto (caption 字段说明)
const CaptionMaxUTF16 = 1024
