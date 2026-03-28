package inspect

import (
	"github.com/maximerivest/mcp2cli/internal/mcp/types"
	"github.com/maximerivest/mcp2cli/internal/naming"
)

// InspectPrompt converts a prompt definition into a ToolSpec-like structure so
// we can reuse CLI rendering and argument parsing.
func InspectPrompt(prompt types.Prompt) *ToolSpec {
	spec := &ToolSpec{
		ToolName:           prompt.Name,
		CLIName:            naming.ToKebabCase(prompt.Name),
		Description:        prompt.Description,
		SupportsCLIParsing: true,
	}
	arguments := make([]ArgSpec, 0, len(prompt.Arguments))
	for _, arg := range prompt.Arguments {
		arguments = append(arguments, ArgSpec{
			Name:        arg.Name,
			CLIName:     naming.ToKebabCase(arg.Name),
			Type:        "string",
			Description: arg.Description,
			Required:    arg.Required,
		})
	}
	spec.Arguments = arguments
	return spec
}
