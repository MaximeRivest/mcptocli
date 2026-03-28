package auth

import (
	"fmt"
	"os"
	"strings"

	"github.com/maximerivest/mcp2cli/internal/config"
	"github.com/maximerivest/mcp2cli/internal/exitcode"
)

// HeadersForServer resolves HTTP headers, including bearer and oauth auth.
func HeadersForServer(store *Store, server *config.Server) (map[string]string, error) {
	headers := map[string]string{}
	if server != nil && server.Headers != nil {
		for key, value := range server.Headers {
			headers[key] = value
		}
	}
	if server == nil {
		return headers, nil
	}

	if server.BearerEnv != "" {
		token := strings.TrimSpace(os.Getenv(server.BearerEnv))
		if token == "" {
			return nil, exitcode.WithHint(exitcode.Newf(exitcode.Auth, "environment variable %q is not set", server.BearerEnv), fmt.Sprintf("set %s before retrying", server.BearerEnv))
		}
		headers["Authorization"] = "Bearer " + token
		return headers, nil
	}

	if strings.EqualFold(server.Auth, "oauth") {
		if authHeader, ok := headers["Authorization"]; ok && authHeader != "" {
			return headers, nil
		}
		if store == nil {
			return nil, exitcode.WithHint(exitcode.New(exitcode.Auth, "oauth token is not available"), loginHint(server))
		}
		token, err := store.Load(TokenKey(server))
		if err != nil || token == nil || strings.TrimSpace(token.AccessToken) == "" {
			return nil, exitcode.WithHint(exitcode.New(exitcode.Auth, "oauth token is not available"), loginHint(server))
		}
		headers["Authorization"] = "Bearer " + token.AccessToken
	}

	return headers, nil
}

func loginHint(server *config.Server) string {
	if server == nil {
		return "run `mcp2cli login <server>`"
	}
	if server.Name != "" && server.Name != "(direct)" {
		return fmt.Sprintf("run `mcp2cli login %s`", server.Name)
	}
	if server.URL != "" {
		return fmt.Sprintf("run `mcp2cli login --url %s --auth oauth`", server.URL)
	}
	return "run `mcp2cli login <server>`"
}
