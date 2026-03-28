package elicitation

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/maximerivest/mcp2cli/internal/mcp/types"
)

func TestHandleObjectSchema(t *testing.T) {
	in := strings.NewReader("y\n")
	out := &bytes.Buffer{}
	handler := NewHandler(in, out)
	result, err := handler.Handle(context.Background(), types.ElicitRequestParams{
		Message: "Do you want to proceed?",
		RequestedSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"confirm": map[string]any{"type": "boolean", "description": "confirm"},
			},
			"required": []string{"confirm"},
		},
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if result.Action != "accept" || result.Content["confirm"] != true {
		t.Fatalf("result = %#v", result)
	}
}

func TestHandleNonInteractiveFile(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "input")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer file.Close()
	handler := NewHandler(file, &bytes.Buffer{})
	_, err = handler.Handle(context.Background(), types.ElicitRequestParams{Message: "Confirm?", RequestedSchema: map[string]any{"type": "object", "properties": map[string]any{"confirm": map[string]any{"type": "boolean"}}, "required": []string{"confirm"}}})
	if err == nil {
		t.Fatal("expected non-interactive error")
	}
}
