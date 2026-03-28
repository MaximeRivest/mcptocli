package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type responseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	for {
		payload, err := readMessage(reader)
		if err != nil {
			if err == io.EOF {
				return
			}
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		var req request
		if err := json.Unmarshal(payload, &req); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		switch req.Method {
		case "initialize":
			respond(req.ID, map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{},
				"serverInfo": map[string]any{
					"name":    "stdiofixture",
					"version": "0.1.0",
				},
			})
		case "notifications/initialized":
			// no-op
		case "tools/list":
			respond(req.ID, map[string]any{
				"tools": []map[string]any{
					{
						"name":        "echo",
						"description": "Echo a message back",
						"inputSchema": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"message": map[string]any{
									"type":        "string",
									"description": "Message to echo",
								},
							},
							"required": []string{"message"},
						},
					},
					{
						"name":        "get-forecast",
						"description": "Get weather forecast for a location",
						"inputSchema": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"latitude": map[string]any{
									"type":        "number",
									"description": "Latitude of the location",
								},
								"longitude": map[string]any{
									"type":        "number",
									"description": "Longitude of the location",
								},
							},
							"required": []string{"latitude", "longitude"},
						},
					},
				},
			})
		case "tools/call":
			var params struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			}
			if err := json.Unmarshal(req.Params, &params); err != nil {
				respondError(req.ID, -32602, fmt.Sprintf("invalid params: %v", err))
				continue
			}
			switch params.Name {
			case "echo":
				message, _ := params.Arguments["message"].(string)
				respond(req.ID, map[string]any{
					"content": []map[string]any{{
						"type": "text",
						"text": "echo: " + message,
					}},
					"structuredContent": map[string]any{
						"message": message,
					},
				})
			case "get-forecast":
				respond(req.ID, map[string]any{
					"content": []map[string]any{{
						"type": "text",
						"text": "Sunny with light winds",
					}},
					"structuredContent": map[string]any{
						"forecast":  "Sunny with light winds",
						"latitude":  params.Arguments["latitude"],
						"longitude": params.Arguments["longitude"],
					},
				})
			default:
				respondError(req.ID, -32601, "unknown tool")
			}
		default:
			respondError(req.ID, -32601, "method not found")
		}
	}
}

func readMessage(reader *bufio.Reader) ([]byte, error) {
	contentLength := -1
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		if line == "\r\n" || line == "\n" {
			break
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(key), "Content-Length") {
			parsed, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return nil, err
			}
			contentLength = parsed
		}
	}
	if contentLength < 0 {
		return nil, fmt.Errorf("missing Content-Length")
	}
	payload := make([]byte, contentLength)
	_, err := io.ReadFull(reader, payload)
	return payload, err
}

func respond(id json.RawMessage, result any) {
	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      rawID(id),
		"result":  result,
	}
	writeMessage(response)
}

func respondError(id json.RawMessage, code int, message string) {
	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      rawID(id),
		"error": responseError{
			Code:    code,
			Message: message,
		},
	}
	writeMessage(response)
}

func rawID(id json.RawMessage) any {
	if len(id) == 0 {
		return nil
	}
	var decoded any
	if err := json.Unmarshal(id, &decoded); err != nil {
		return nil
	}
	return decoded
}

func writeMessage(message any) {
	payload, err := json.Marshal(message)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Content-Length: %d\r\n\r\n", len(payload))
	if _, err := os.Stdout.Write(payload); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
