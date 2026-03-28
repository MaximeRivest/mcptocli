package inspect

import (
	"testing"

	"github.com/maximerivest/mcp2cli/internal/mcp/types"
)

func TestInspectPrompt(t *testing.T) {
	spec := InspectPrompt(types.Prompt{
		Name:        "review-code",
		Description: "Generate a review prompt",
		Arguments:   []types.PromptArgument{{Name: "code", Required: true}, {Name: "focus"}},
	})
	if spec.CLIName != "review-code" {
		t.Fatalf("CLIName = %q", spec.CLIName)
	}
	if len(spec.Arguments) != 2 || spec.Arguments[0].CLIName != "code" {
		t.Fatalf("Arguments = %#v", spec.Arguments)
	}
}
