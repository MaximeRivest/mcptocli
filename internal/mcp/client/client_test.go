package client

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/maximerivest/mcp2cli/internal/config"
)

func TestConnectStdioListAndCallTool(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	fixtureDir := filepath.Join(repoRoot, "testdata", "servers", "stdiofixture")
	command := fmt.Sprintf("go run %q", fixtureDir)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := ConnectStdio(ctx, &config.Server{Command: command})
	if err != nil {
		t.Fatalf("ConnectStdio: %v", err)
	}
	defer func() { _ = client.Close() }()

	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools) < 2 {
		t.Fatalf("expected at least 2 tools, got %d", len(tools))
	}

	result, err := client.CallTool(ctx, "echo", map[string]any{"message": "hello"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if len(result.Content) != 1 || result.Content[0].Text != "echo: hello" {
		t.Fatalf("unexpected tool result: %#v", result)
	}
	if result.StructuredContent["message"] != "hello" {
		t.Fatalf("unexpected structured content: %#v", result.StructuredContent)
	}
	resources, err := client.ListResources(ctx)
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(resources))
	}
	resource, err := client.ReadResource(ctx, "resource://docs/api")
	if err != nil {
		t.Fatalf("ReadResource: %v", err)
	}
	if len(resource.Contents) != 1 || resource.Contents[0].Text != "API docs for the weather service" {
		t.Fatalf("unexpected resource result: %#v", resource)
	}
	prompts, err := client.ListPrompts(ctx)
	if err != nil {
		t.Fatalf("ListPrompts: %v", err)
	}
	if len(prompts) != 1 || prompts[0].Name != "review-code" {
		t.Fatalf("unexpected prompts: %#v", prompts)
	}
	prompt, err := client.GetPrompt(ctx, "review-code", map[string]string{"code": "x", "focus": "api"})
	if err != nil {
		t.Fatalf("GetPrompt: %v", err)
	}
	if len(prompt.Messages) != 1 || prompt.Messages[0].Content.Text == "" {
		t.Fatalf("unexpected prompt result: %#v", prompt)
	}
}
