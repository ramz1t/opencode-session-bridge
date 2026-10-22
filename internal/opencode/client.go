package opencode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL  string
	username string
	password string
	http     *http.Client
}

type Session struct {
	ID        string
	Title     string
	Directory string
	Outcome   string
	Created   time.Time
	Updated   time.Time
}

type Message struct {
	ID      string
	Role    string
	Text    string
	Created time.Time
}

type rawSession struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Outcome  string `json:"outcome"`
	Location struct {
		Directory string `json:"directory"`
	} `json:"location"`
	Time struct {
		Created int64 `json:"created"`
		Updated int64 `json:"updated"`
	} `json:"time"`
}

type rawMessage struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Time struct {
		Created int64 `json:"created"`
	} `json:"time"`
	Text    string `json:"text"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

func NewClient(cfg Config) *Client {
	return &Client{
		baseURL:  strings.TrimRight(cfg.BaseURL, "/"),
		username: cfg.Username,
		password: cfg.Password,
		http:     &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) ListSessions(ctx context.Context, limit int) ([]Session, error) {
	query := url.Values{}
	query.Set("limit", strconv.Itoa(limit))
	query.Set("order", "desc")

	var page struct {
		Data []rawSession `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/session?"+query.Encode(), nil, &page); err != nil {
		return nil, err
	}

	sessions := make([]Session, 0, len(page.Data))
	for _, r := range page.Data {
		sessions = append(sessions, Session{
			ID:        r.ID,
			Title:     r.Title,
			Directory: r.Location.Directory,
			Outcome:   r.Outcome,
			Created:   time.UnixMilli(r.Time.Created),
			Updated:   time.UnixMilli(r.Time.Updated),
		})
	}
	return sessions, nil
}

func (c *Client) ActiveSessionIDs(ctx context.Context) (map[string]bool, error) {
	var body struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/session/active", nil, &body); err != nil {
		return nil, err
	}

	active := make(map[string]bool, len(body.Data))
	for id := range body.Data {
		active[id] = true
	}
	return active, nil
}

// SendPrompt returns the ID of the queued inbox item.
func (c *Client) SendPrompt(ctx context.Context, sessionID, text, delivery string) (string, error) {
	body := map[string]any{"text": text}
	if delivery != "" {
		body["delivery"] = delivery
	}

	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := c.do(ctx, http.MethodPost, sessionPath(sessionID)+"/prompt", body, &resp); err != nil {
		return "", err
	}
	return resp.Data.ID, nil
}

// Messages returns user and assistant messages, newest first.
func (c *Client) Messages(ctx context.Context, sessionID string, limit int) ([]Message, error) {
	query := url.Values{}
	query.Set("limit", strconv.Itoa(limit))
	query.Set("order", "desc")

	var page struct {
		Data []rawMessage `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, sessionPath(sessionID)+"/message?"+query.Encode(), nil, &page); err != nil {
		return nil, err
	}

	messages := make([]Message, 0, len(page.Data))
	for _, r := range page.Data {
		switch r.Type {
		case "user":
			messages = append(messages, Message{ID: r.ID, Role: "user", Text: r.Text, Created: time.UnixMilli(r.Time.Created)})
		case "assistant":
			messages = append(messages, Message{ID: r.ID, Role: "assistant", Text: assistantText(r), Created: time.UnixMilli(r.Time.Created)})
		}
	}
	return messages, nil
}

func assistantText(r rawMessage) string {
	var parts []string
	for _, part := range r.Content {
		if part.Type == "text" && part.Text != "" {
			parts = append(parts, part.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func sessionPath(sessionID string) string {
	return "/api/session/" + url.PathEscape(sessionID)
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("opencode request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("opencode %s %s returned %s: %s", method, path, resp.Status, strings.TrimSpace(string(data)))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(data, out)
}
