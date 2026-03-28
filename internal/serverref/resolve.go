package serverref

import (
	"fmt"
	"strings"

	"github.com/maximerivest/mcp2cli/internal/config"
)

// Resolved describes a server selected from config or ephemeral CLI flags.
type Resolved struct {
	DisplayName string
	Server      *config.Server
}

// Resolve selects the effective server for a command.
func Resolve(repo *config.Repository, bound *config.Server, explicitName, command, url, cwd string, envVars []string) (*Resolved, error) {
	if command != "" && url != "" {
		return nil, fmt.Errorf("--command and --url are mutually exclusive")
	}
	if bound != nil && (explicitName != "" || command != "" || url != "") {
		return nil, fmt.Errorf("bound exposed commands do not accept an explicit server, --command, or --url")
	}

	if bound != nil {
		server := cloneServer(bound)
		if err := applyOverrides(server, cwd, envVars); err != nil {
			return nil, err
		}
		return &Resolved{DisplayName: server.Name, Server: server}, nil
	}

	if command != "" || url != "" {
		server := &config.Server{
			Name:    "(direct)",
			Source:  config.SourceEphemeral,
			Command: strings.TrimSpace(command),
			URL:     strings.TrimSpace(url),
		}
		if err := applyOverrides(server, cwd, envVars); err != nil {
			return nil, err
		}
		display := server.Command
		if server.URL != "" {
			display = server.URL
		}
		return &Resolved{DisplayName: display, Server: server}, nil
	}

	if explicitName == "" {
		return nil, fmt.Errorf("a registered server name or one of --command/--url is required")
	}

	server, err := repo.ResolveServer(explicitName)
	if err != nil {
		return nil, err
	}
	if err := applyOverrides(server, cwd, envVars); err != nil {
		return nil, err
	}
	return &Resolved{DisplayName: server.Name, Server: server}, nil
}

func applyOverrides(server *config.Server, cwd string, envVars []string) error {
	if cwd != "" {
		server.CWD = cwd
	}
	if len(envVars) == 0 {
		return nil
	}
	if server.Env == nil {
		server.Env = map[string]string{}
	}
	for _, entry := range envVars {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			return fmt.Errorf("invalid --env value %q, expected KEY=VALUE", entry)
		}
		server.Env[key] = value
	}
	return nil
}

func cloneServer(server *config.Server) *config.Server {
	if server == nil {
		return &config.Server{}
	}
	clone := *server
	if server.Env != nil {
		clone.Env = make(map[string]string, len(server.Env))
		for k, v := range server.Env {
			clone.Env[k] = v
		}
	}
	if server.Headers != nil {
		clone.Headers = make(map[string]string, len(server.Headers))
		for k, v := range server.Headers {
			clone.Headers[k] = v
		}
	}
	if server.Roots != nil {
		clone.Roots = append([]string(nil), server.Roots...)
	}
	if server.ExposeAs != nil {
		clone.ExposeAs = append([]string(nil), server.ExposeAs...)
	}
	return &clone
}
