package types

// Resource is an MCP resource descriptor.
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ListResourcesResult is returned by resources/list.
type ListResourcesResult struct {
	Resources  []Resource `json:"resources"`
	NextCursor string     `json:"nextCursor,omitempty"`
}

// ReadResourceParams is sent to resources/read.
type ReadResourceParams struct {
	URI string `json:"uri"`
}

// ResourceContent is one resource content block.
type ResourceContent struct {
	URI      string `json:"uri,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
}

// ReadResourceResult is returned by resources/read.
type ReadResourceResult struct {
	Contents []ResourceContent `json:"contents,omitempty"`
}

// PromptArgument describes one prompt argument.
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// Prompt is an MCP prompt descriptor.
type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// ListPromptsResult is returned by prompts/list.
type ListPromptsResult struct {
	Prompts    []Prompt `json:"prompts"`
	NextCursor string   `json:"nextCursor,omitempty"`
}

// GetPromptParams is sent to prompts/get.
type GetPromptParams struct {
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

// PromptMessage is one prompt message block.
type PromptMessage struct {
	Role    string  `json:"role"`
	Content Content `json:"content"`
}

// GetPromptResult is returned by prompts/get.
type GetPromptResult struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages,omitempty"`
}

// ElicitRequestParams is sent by the server when it needs user input.
type ElicitRequestParams struct {
	Message         string         `json:"message"`
	RequestedSchema map[string]any `json:"requestedSchema"`
}

// ElicitResult is returned by the client in response to elicitation/create.
type ElicitResult struct {
	Action  string         `json:"action"`
	Content map[string]any `json:"content,omitempty"`
}
