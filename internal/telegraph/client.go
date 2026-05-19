// Package telegraph fetches and renders Telegra.ph articles via the public
// https://api.telegra.ph/getPage endpoint.
package telegraph

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	apiBaseURL   = "https://api.telegra.ph"
	defaultHost  = "telegra.ph"
	fetchTimeout = 30 * time.Second
)

// Client calls the Telegra.ph getPage API.
type Client struct {
	http *http.Client
}

// NewClient creates a client. httpClient may carry the bot's configured proxy;
// when nil, http.DefaultClient is used with a per-request timeout via context.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: fetchTimeout}
	}
	return &Client{http: httpClient}
}

// Page is the subset of the Telegra.ph Page object we care about.
type Page struct {
	Path        string `json:"path"`
	Title       string `json:"title"`
	AuthorName  string `json:"author_name"`
	Description string `json:"description"`
	Content     []Node `json:"content"`
}

// Node is a DOM node that is either a raw string (text) or an element with a
// tag, optional attributes, and optional children.
//
// The Telegra.ph API encodes text nodes as bare JSON strings and element nodes
// as objects, so we implement UnmarshalJSON to dispatch on the token type.
type Node struct {
	Text     string
	Tag      string
	Attrs    map[string]string
	Children []Node
}

// IsText reports whether this node is a raw text node (no tag).
func (n *Node) IsText() bool { return n.Tag == "" }

// UnmarshalJSON distinguishes string nodes from element objects.
func (n *Node) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if len(trimmed) == 0 {
		return nil
	}
	if trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		n.Text = s
		return nil
	}

	var raw struct {
		Tag      string            `json:"tag"`
		Attrs    map[string]string `json:"attrs"`
		Children []Node            `json:"children"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	n.Tag = raw.Tag
	n.Attrs = raw.Attrs
	n.Children = raw.Children
	return nil
}

// apiResponse is the envelope returned by every Telegra.ph API endpoint.
type apiResponse struct {
	OK     bool            `json:"ok"`
	Error  string          `json:"error"`
	Result json.RawMessage `json:"result"`
}

// GetPage fetches and parses the article at the given telegra.ph URL or path.
// Accepts full URLs (https://telegra.ph/slug) or bare paths ("slug").
func (c *Client) GetPage(ctx context.Context, pathOrURL string) (*Page, error) {
	path, err := ExtractPath(pathOrURL)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/getPage/%s?return_content=true", apiBaseURL, url.PathEscape(path))

	reqCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request telegraph api: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("telegraph api status %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var env apiResponse
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("decode envelope: %w", err)
	}
	if !env.OK {
		return nil, fmt.Errorf("telegraph api error: %s", env.Error)
	}

	var page Page
	if err := json.Unmarshal(env.Result, &page); err != nil {
		return nil, fmt.Errorf("decode page: %w", err)
	}
	return &page, nil
}

// ExtractPath validates that the input points to a telegra.ph article and
// returns its path segment (e.g. "MK-test-telegraph-dev-05-10").
// It accepts bare paths too, so callers may feed either a full URL or a slug.
func ExtractPath(input string) (string, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", fmt.Errorf("empty input")
	}

	// Bare path: no scheme, no dot → treat as slug.
	if !strings.Contains(s, "://") && !strings.Contains(s, ".") {
		return strings.Trim(s, "/"), nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("invalid url: %w", err)
	}
	if u.Host != defaultHost && !strings.HasSuffix(u.Host, "."+defaultHost) {
		return "", fmt.Errorf("unsupported host %q, expected telegra.ph", u.Host)
	}
	path := strings.Trim(u.Path, "/")
	if path == "" {
		return "", fmt.Errorf("missing article path")
	}
	return path, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
