package invoke

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/maximerivest/mcp2cli/internal/exitcode"
	"github.com/maximerivest/mcp2cli/internal/schema/inspect"
)

// ParseToolArguments converts runtime CLI tokens into a tool arguments object.
func ParseToolArguments(spec *inspect.ToolSpec, tokens []string) (map[string]any, error) {
	if spec == nil {
		return map[string]any{}, nil
	}
	if !spec.SupportsCLIParsing {
		if len(tokens) == 0 {
			return applyDefaults(spec, map[string]any{}), nil
		}
		return nil, exitcode.Newf(exitcode.Usage, "tool %q requires --input because its schema is too complex for flags and positionals", spec.CLIName)
	}

	result := map[string]any{}
	positional := spec.PositionalArguments()
	positionalIndex := 0
	seenFlag := false

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		if token == "--" {
			for i = i + 1; i < len(tokens); i++ {
				if positionalIndex >= len(positional) {
					return nil, exitcode.Newf(exitcode.Usage, "unexpected positional argument %q", tokens[i])
				}
				arg := positional[positionalIndex]
				value, err := parseValue(arg, tokens[i])
				if err != nil {
					return nil, exitcode.Wrapf(exitcode.Usage, err, "parse %s", arg.CLIName)
				}
				result[arg.Name] = value
				positionalIndex++
			}
			break
		}

		if strings.HasPrefix(token, "--") {
			seenFlag = true
			name := strings.TrimPrefix(token, "--")
			valueText := ""
			hasInline := false

			if strings.HasPrefix(name, "no-") {
				arg, ok := spec.FindArgument(strings.TrimPrefix(name, "no-"))
				if !ok || arg.Type != "boolean" {
					return nil, exitcode.Newf(exitcode.Usage, "unknown flag --%s", name)
				}
				result[arg.Name] = false
				continue
			}
			if before, after, ok := strings.Cut(name, "="); ok {
				name = before
				valueText = after
				hasInline = true
			}
			arg, ok := spec.FindArgument(name)
			if !ok {
				return nil, exitcode.Newf(exitcode.Usage, "unknown flag --%s", name)
			}

			if arg.Type == "boolean" && !hasInline {
				if i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "-") {
					i++
					parsed, err := strconv.ParseBool(tokens[i])
					if err != nil {
						return nil, exitcode.Wrapf(exitcode.Usage, err, "parse --%s", name)
					}
					result[arg.Name] = parsed
				} else {
					result[arg.Name] = true
				}
				continue
			}
			if !hasInline {
				i++
				if i >= len(tokens) {
					return nil, exitcode.Newf(exitcode.Usage, "missing value for --%s", name)
				}
				valueText = tokens[i]
			}
			value, err := parseValue(arg, valueText)
			if err != nil {
				return nil, exitcode.Wrapf(exitcode.Usage, err, "parse --%s", name)
			}
			if arg.Type == "array" {
				existing, _ := result[arg.Name].([]any)
				result[arg.Name] = append(existing, value)
			} else {
				result[arg.Name] = value
			}
			continue
		}

		if seenFlag {
			return nil, exitcode.Newf(exitcode.Usage, "unexpected positional argument %q after flags", token)
		}
		if positionalIndex >= len(positional) {
			return nil, exitcode.Newf(exitcode.Usage, "unexpected positional argument %q", token)
		}
		arg := positional[positionalIndex]
		value, err := parseValue(arg, token)
		if err != nil {
			return nil, exitcode.Wrapf(exitcode.Usage, err, "parse %s", arg.CLIName)
		}
		result[arg.Name] = value
		positionalIndex++
	}

	result = applyDefaults(spec, result)
	for _, arg := range spec.Arguments {
		if arg.Required {
			if _, ok := result[arg.Name]; !ok {
				return nil, exitcode.Newf(exitcode.Usage, "missing required argument: %s", arg.CLIName)
			}
		}
	}

	return result, nil
}

func parseValue(arg inspect.ArgSpec, text string) (any, error) {
	switch arg.Type {
	case "string":
		if strings.HasPrefix(text, "@") {
			data, err := ReadAtValue(text)
			if err != nil {
				return nil, err
			}
			return string(data), nil
		}
		return text, nil
	case "integer":
		value, err := strconv.Atoi(text)
		if err != nil {
			return nil, err
		}
		return value, nil
	case "number":
		value, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return nil, err
		}
		return value, nil
	case "boolean":
		value, err := strconv.ParseBool(text)
		if err != nil {
			return nil, err
		}
		return value, nil
	case "object":
		data, err := parseStructuredInput(text)
		if err != nil {
			return nil, err
		}
		var value map[string]any
		if err := json.Unmarshal(data, &value); err != nil {
			return nil, err
		}
		return value, nil
	case "array":
		if arg.ItemType == "" {
			data, err := parseStructuredInput(text)
			if err != nil {
				return nil, err
			}
			var value []any
			if err := json.Unmarshal(data, &value); err != nil {
				return nil, err
			}
			return value, nil
		}
		return parseValue(inspect.ArgSpec{Type: arg.ItemType}, text)
	default:
		return nil, fmt.Errorf("unsupported argument type %q", arg.Type)
	}
}

func parseStructuredInput(text string) ([]byte, error) {
	if strings.HasPrefix(text, "@") {
		return ReadAtValue(text)
	}
	return []byte(text), nil
}

// ReadAtValue loads @file or @- style CLI values.
func ReadAtValue(text string) ([]byte, error) {
	name := strings.TrimPrefix(text, "@")
	if name == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(name)
}

func applyDefaults(spec *inspect.ToolSpec, arguments map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range arguments {
		result[key] = value
	}
	for _, arg := range spec.Arguments {
		if _, ok := result[arg.Name]; ok {
			continue
		}
		if arg.HasDefault {
			result[arg.Name] = arg.Default
		}
	}
	return result
}
