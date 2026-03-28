package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/maximerivest/mcp2cli/internal/config"
)

func TestConnectHTTPListAndCallTool(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Decode request: %v", err)
		}
		method, _ := req["method"].(string)
		id := req["id"]
		w.Header().Set("Content-Type", "application/json")
		switch method {
		case "initialize":
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"protocolVersion": "2024-11-05", "capabilities": map[string]any{}, "serverInfo": map[string]any{"name": "httpfixture", "version": "0.1.0"}}})
		case "notifications/initialized":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","result":{}}`))
		case "tools/list":
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"tools": []map[string]any{{"name": "echo", "description": "Echo a message back"}}}})
		case "tools/call":
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"content": []map[string]any{{"type": "text", "text": "echo: hello"}}}})
		case "resources/list":
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"resources": []map[string]any{{"uri": "resource://docs/api", "name": "api-docs"}}}})
		case "resources/read":
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"contents": []map[string]any{{"uri": "resource://docs/api", "text": "API docs"}}}})
		case "prompts/list":
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"prompts": []map[string]any{{"name": "review-code", "arguments": []map[string]any{{"name": "code", "required": true}}}}}})
		case "prompts/get":
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"messages": []map[string]any{{"role": "user", "content": map[string]any{"type": "text", "text": "Review this code"}}}}})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": -32601, "message": "method not found"}})
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := ConnectHTTP(ctx, &config.Server{URL: server.URL}, nil)
	if err != nil {
		t.Fatalf("ConnectHTTP: %v", err)
	}
	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "echo" {
		t.Fatalf("tools = %#v", tools)
	}
	result, err := client.CallTool(ctx, "echo", map[string]any{"message": "hello"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if len(result.Content) != 1 || result.Content[0].Text != "echo: hello" {
		t.Fatalf("result = %#v", result)
	}
	resources, err := client.ListResources(ctx)
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}
	if len(resources) != 1 || resources[0].URI != "resource://docs/api" {
		t.Fatalf("resources = %#v", resources)
	}
	resource, err := client.ReadResource(ctx, "resource://docs/api")
	if err != nil {
		t.Fatalf("ReadResource: %v", err)
	}
	if len(resource.Contents) != 1 || resource.Contents[0].Text != "API docs" {
		t.Fatalf("resource = %#v", resource)
	}
	prompts, err := client.ListPrompts(ctx)
	if err != nil {
		t.Fatalf("ListPrompts: %v", err)
	}
	if len(prompts) != 1 || prompts[0].Name != "review-code" {
		t.Fatalf("prompts = %#v", prompts)
	}
	prompt, err := client.GetPrompt(ctx, "review-code", map[string]string{"code": "x"})
	if err != nil {
		t.Fatalf("GetPrompt: %v", err)
	}
	if len(prompt.Messages) != 1 || prompt.Messages[0].Content.Text != "Review this code" {
		t.Fatalf("prompt = %#v", prompt)
	}
}
