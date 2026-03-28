package elicitation

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/maximerivest/mcp2cli/internal/exitcode"
	"github.com/maximerivest/mcp2cli/internal/mcp/types"
	"github.com/maximerivest/mcp2cli/internal/schema/inspect"
	"golang.org/x/term"
)

// Handler prompts the user in the terminal for elicitation requests.
type Handler struct {
	In  io.Reader
	Out io.Writer
}

// NewHandler constructs a terminal elicitation handler.
func NewHandler(in io.Reader, out io.Writer) *Handler {
	return &Handler{In: in, Out: out}
}

// Handle services one elicitation request from the server.
func (h *Handler) Handle(ctx context.Context, params types.ElicitRequestParams) (*types.ElicitResult, error) {
	_ = ctx
	if h == nil {
		return nil, exitcode.New(exitcode.Interactive, "elicitation is not available")
	}
	if !isInteractive(h.In) {
		return nil, exitcode.WithHint(exitcode.New(exitcode.Interactive, "server requested user input, but stdin is not interactive"), "run this command in an interactive terminal")
	}

	if h.Out == nil {
		h.Out = io.Discard
	}
	fmt.Fprintf(h.Out, "\nServer asks: %s\n", strings.TrimSpace(params.Message))

	schemaBytes, err := json.Marshal(params.RequestedSchema)
	if err != nil {
		return nil, exitcode.Wrap(exitcode.Internal, err, "encode elicitation schema")
	}
	spec, err := inspect.InspectTool(types.Tool{Name: "elicitation", InputSchema: schemaBytes})
	if err != nil {
		return nil, exitcode.Wrap(exitcode.Internal, err, "inspect elicitation schema")
	}

	reader := bufio.NewReader(h.In)
	content := map[string]any{}
	for _, arg := range spec.Arguments {
		value, ok, err := promptValue(reader, h.Out, arg)
		if err != nil {
			return nil, err
		}
		if ok {
			content[arg.Name] = value
		}
	}
	return &types.ElicitResult{Action: "accept", Content: content}, nil
}

func promptValue(reader *bufio.Reader, out io.Writer, arg inspect.ArgSpec) (any, bool, error) {
	for {
		label := arg.CLIName
		if arg.Description != "" {
			label = arg.Description
		}
		required := ""
		if !arg.Required {
			required = " (optional)"
		}
		fmt.Fprintf(out, "%s%s: ", label, required)
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, false, exitcode.Wrap(exitcode.Interactive, err, "read elicitation input")
		}
		line = strings.TrimSpace(line)
		if line == "" {
			if arg.Required {
				fmt.Fprintln(out, "Value is required.")
				continue
			}
			return nil, false, nil
		}
		switch arg.Type {
		case "boolean":
			if strings.EqualFold(line, "y") || strings.EqualFold(line, "yes") {
				return true, true, nil
			}
			if strings.EqualFold(line, "n") || strings.EqualFold(line, "no") {
				return false, true, nil
			}
			value, err := strconv.ParseBool(line)
			if err != nil {
				fmt.Fprintln(out, "Enter y/n or true/false.")
				continue
			}
			return value, true, nil
		case "integer":
			value, err := strconv.Atoi(line)
			if err != nil {
				fmt.Fprintln(out, "Enter an integer.")
				continue
			}
			return value, true, nil
		case "number":
			value, err := strconv.ParseFloat(line, 64)
			if err != nil {
				fmt.Fprintln(out, "Enter a number.")
				continue
			}
			return value, true, nil
		default:
			return line, true, nil
		}
	}
}

func isInteractive(in io.Reader) bool {
	file, ok := in.(*os.File)
	if !ok {
		return true
	}
	return term.IsTerminal(int(file.Fd()))
}
