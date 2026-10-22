package bridge

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"opencode-session-bridge/internal/opencode"
)

// callerSessionKey is the _meta key OpenCode sets on tool calls made from a session.
const callerSessionKey = "ai.opencode/sessionID"

const (
	defaultListLimit = 10
	maxListLimit     = 50
	discoveryLimit   = 200
	defaultReadLimit = 5
	maxReadLimit     = 50
	maxWaitSeconds   = 900
	replyPollEvery   = 2 * time.Second
)

// Provider returns a client connected to the local OpenCode service.
type Provider func(ctx context.Context) (*opencode.Client, error)

type ListSessionsInput struct {
	Project        string `json:"project,omitempty" jsonschema:"Project name or path fragment such as my-app. Matches a directory path segment first, then any substring of the path."`
	Title          string `json:"title,omitempty" jsonschema:"Case-insensitive text that the session title must contain."`
	ActiveOnly     bool   `json:"active_only,omitempty" jsonschema:"Only return sessions that are running right now."`
	IncludeCurrent bool   `json:"include_current,omitempty" jsonschema:"Include the session that is calling this tool. Off by default."`
	Limit          int    `json:"limit,omitempty" jsonschema:"Maximum number of sessions to return. Defaults to 10, maximum 50."`
}

type ListSessionsOutput struct {
	Sessions []SessionView `json:"sessions"`
}

type FindSessionInput struct {
	Project string `json:"project" jsonschema:"Project name or path fragment such as my-app. The most recently active matching session is returned."`
	Title   string `json:"title,omitempty" jsonschema:"Optional case-insensitive text the session title must contain."`
}

type SendMessageInput struct {
	SessionID   string `json:"session_id" jsonschema:"ID of the target session, starting with ses_. Get it from list_sessions or find_session."`
	Text        string `json:"text" jsonschema:"The message to send to the session."`
	Delivery    string `json:"delivery,omitempty" jsonschema:"steer to interrupt the current work with this message, or queue to deliver it after the current work. Defaults to the server's choice."`
	WaitSeconds int    `json:"wait_seconds,omitempty" jsonschema:"Seconds to wait for the session to finish and reply. 0 returns right after the message is queued. Maximum 900."`
}

type SendMessageOutput struct {
	SessionID string `json:"session_id"`
	InboxID   string `json:"inbox_id"`
	Replied   bool   `json:"replied"`
	Running   bool   `json:"running"`
	Reply     string `json:"reply,omitempty"`
}

type ReadSessionInput struct {
	SessionID string `json:"session_id" jsonschema:"ID of the session to read, starting with ses_."`
	Limit     int    `json:"limit,omitempty" jsonschema:"Number of recent user and assistant messages to return. Defaults to 5, maximum 50."`
}

type ReadSessionOutput struct {
	SessionID string        `json:"session_id"`
	Running   bool          `json:"running"`
	Messages  []MessageView `json:"messages"`
}

type SessionView struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Directory  string `json:"directory"`
	Running    bool   `json:"running"`
	LastActive string `json:"last_active"`
	Outcome    string `json:"outcome,omitempty"`
}

type MessageView struct {
	Role    string `json:"role"`
	Text    string `json:"text"`
	Created string `json:"created"`
}

func Register(server *mcp.Server, provider Provider) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_sessions",
		Description: "List OpenCode sessions, most recently active first. Filter by project or title, or only running sessions. Use this to discover session IDs before sending a message.",
	}, listSessions(provider))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "find_session",
		Description: "Return the most recently active OpenCode session for a project, such as 'the last active session in my-app'. Running sessions are preferred. The calling session is never returned.",
	}, findSession(provider))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "send_message",
		Description: "Send a message to another OpenCode session. Set wait_seconds to block until the session replies and get the reply back. The calling session cannot be the target.",
	}, sendMessage(provider))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "read_session",
		Description: "Read the most recent user and assistant messages of an OpenCode session, oldest first, and whether it is still running.",
	}, readSession(provider))
}

func listSessions(provider Provider) mcp.ToolHandlerFor[ListSessionsInput, ListSessionsOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in ListSessionsInput) (*mcp.CallToolResult, ListSessionsOutput, error) {
		client, err := provider(ctx)
		if err != nil {
			return nil, ListSessionsOutput{}, err
		}
		sessions, running, err := load(ctx, client)
		if err != nil {
			return nil, ListSessionsOutput{}, err
		}

		exclude := ""
		if !in.IncludeCurrent {
			exclude = callerSession(req)
		}
		ranked := Rank(sessions, running, Filter{
			Project:    in.Project,
			Title:      in.Title,
			ActiveOnly: in.ActiveOnly,
			Exclude:    exclude,
		})

		limit := clamp(in.Limit, defaultListLimit, maxListLimit)
		if len(ranked) > limit {
			ranked = ranked[:limit]
		}
		return nil, ListSessionsOutput{Sessions: toViews(ranked, running)}, nil
	}
}

func findSession(provider Provider) mcp.ToolHandlerFor[FindSessionInput, SessionView] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in FindSessionInput) (*mcp.CallToolResult, SessionView, error) {
		if strings.TrimSpace(in.Project) == "" {
			return nil, SessionView{}, errors.New("project is required")
		}
		client, err := provider(ctx)
		if err != nil {
			return nil, SessionView{}, err
		}
		sessions, running, err := load(ctx, client)
		if err != nil {
			return nil, SessionView{}, err
		}

		ranked := Rank(sessions, running, Filter{
			Project: in.Project,
			Title:   in.Title,
			Exclude: callerSession(req),
		})
		if len(ranked) == 0 {
			return nil, SessionView{}, fmt.Errorf("no session found for project %q among the %d most recent sessions", in.Project, len(sessions))
		}
		return nil, toViews(ranked[:1], running)[0], nil
	}
}

func sendMessage(provider Provider) mcp.ToolHandlerFor[SendMessageInput, SendMessageOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in SendMessageInput) (*mcp.CallToolResult, SendMessageOutput, error) {
		if in.SessionID == "" || strings.TrimSpace(in.Text) == "" {
			return nil, SendMessageOutput{}, errors.New("session_id and text are required")
		}
		if in.SessionID == callerSession(req) {
			return nil, SendMessageOutput{}, errors.New("refusing to send a message to the calling session")
		}
		if in.Delivery != "" && in.Delivery != "steer" && in.Delivery != "queue" {
			return nil, SendMessageOutput{}, errors.New(`delivery must be "steer" or "queue"`)
		}
		client, err := provider(ctx)
		if err != nil {
			return nil, SendMessageOutput{}, err
		}

		sentAt := time.Now()
		inboxID, err := client.SendPrompt(ctx, in.SessionID, in.Text, in.Delivery)
		if err != nil {
			return nil, SendMessageOutput{}, err
		}
		out := SendMessageOutput{SessionID: in.SessionID, InboxID: inboxID}
		if in.WaitSeconds <= 0 {
			return nil, out, nil
		}

		wait := time.Duration(min(in.WaitSeconds, maxWaitSeconds)) * time.Second
		reply, replied, running, err := waitForReply(ctx, client, in.SessionID, sentAt, wait)
		if err != nil {
			return nil, out, err
		}
		out.Reply = reply
		out.Replied = replied
		out.Running = running
		return nil, out, nil
	}
}

func readSession(provider Provider) mcp.ToolHandlerFor[ReadSessionInput, ReadSessionOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in ReadSessionInput) (*mcp.CallToolResult, ReadSessionOutput, error) {
		if in.SessionID == "" {
			return nil, ReadSessionOutput{}, errors.New("session_id is required")
		}
		client, err := provider(ctx)
		if err != nil {
			return nil, ReadSessionOutput{}, err
		}

		messages, err := client.Messages(ctx, in.SessionID, clamp(in.Limit, defaultReadLimit, maxReadLimit))
		if err != nil {
			return nil, ReadSessionOutput{}, err
		}
		active, err := client.ActiveSessionIDs(ctx)
		if err != nil {
			return nil, ReadSessionOutput{}, err
		}

		views := make([]MessageView, 0, len(messages))
		for i := len(messages) - 1; i >= 0; i-- {
			views = append(views, toMessageView(messages[i]))
		}
		return nil, ReadSessionOutput{SessionID: in.SessionID, Running: active[in.SessionID], Messages: views}, nil
	}
}

// waitForReply polls until the session has stopped running and has an
// assistant message created after sentAt, or until the wait expires.
func waitForReply(ctx context.Context, client *opencode.Client, sessionID string, sentAt time.Time, wait time.Duration) (reply string, replied bool, running bool, err error) {
	deadline := time.Now().Add(wait)
	for {
		messages, err := client.Messages(ctx, sessionID, 10)
		if err != nil {
			return "", false, false, err
		}
		active, err := client.ActiveSessionIDs(ctx)
		if err != nil {
			return "", false, false, err
		}
		running = active[sessionID]

		reply = latestAssistantSince(messages, sentAt)
		if reply != "" && !running {
			return reply, true, false, nil
		}
		if time.Now().After(deadline) {
			return reply, false, running, nil
		}

		select {
		case <-ctx.Done():
			return reply, false, running, ctx.Err()
		case <-time.After(replyPollEvery):
		}
	}
}

func latestAssistantSince(messages []opencode.Message, since time.Time) string {
	for _, m := range messages {
		if m.Role == "assistant" && !m.Created.Before(since) && m.Text != "" {
			return m.Text
		}
	}
	return ""
}

func load(ctx context.Context, client *opencode.Client) ([]opencode.Session, map[string]bool, error) {
	sessions, err := client.ListSessions(ctx, discoveryLimit)
	if err != nil {
		return nil, nil, err
	}
	running, err := client.ActiveSessionIDs(ctx)
	if err != nil {
		return nil, nil, err
	}
	return sessions, running, nil
}

func callerSession(req *mcp.CallToolRequest) string {
	if req == nil || req.Params == nil {
		return ""
	}
	id, _ := req.Params.Meta[callerSessionKey].(string)
	return id
}

func toViews(sessions []opencode.Session, running map[string]bool) []SessionView {
	views := make([]SessionView, 0, len(sessions))
	for _, s := range sessions {
		views = append(views, SessionView{
			ID:         s.ID,
			Title:      s.Title,
			Directory:  s.Directory,
			Running:    running[s.ID],
			LastActive: s.Updated.Format(time.RFC3339),
			Outcome:    s.Outcome,
		})
	}
	return views
}

func toMessageView(m opencode.Message) MessageView {
	return MessageView{Role: m.Role, Text: m.Text, Created: m.Created.Format(time.RFC3339)}
}

func clamp(value, def, ceiling int) int {
	if value <= 0 {
		return def
	}
	return min(value, ceiling)
}
