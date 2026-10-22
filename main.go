package main

import (
	"context"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"opencode-session-bridge/internal/bridge"
	"opencode-session-bridge/internal/opencode"
)

var version = "dev"

func main() {
	log.SetOutput(os.Stderr)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "opencode-session-bridge",
		Version: version,
	}, nil)

	bridge.Register(server, connect)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

// connect discovers the local OpenCode service on every call, so the bridge
// keeps working after OpenCode restarts on a different port.
func connect(ctx context.Context) (*opencode.Client, error) {
	cfg, err := opencode.Discover(ctx)
	if err != nil {
		return nil, err
	}
	return opencode.NewClient(cfg), nil
}
