package serverref

import (
	"strings"

	"github.com/maximerivest/mcp2cli/internal/config"
	"github.com/maximerivest/mcp2cli/internal/exitcode"
)

// Options contains CLI-level overrides used to resolve a server reference.
type Options struct {
	ExplicitName      string
	Command           string
	URL               string
	CWD               string
	Env               []string
	Headers           []string
	Auth              string
	BearerEnv         string
	OAuthAuthorizeURL string
	OAuthTokenURL     string
	OAuthClientID     string
	OAuthScopes       []string
}

// Resolved describes a server selected from config or ephemeral CLI flags.
type Resolved struct {
	DisplayName string
	Server      *config.Server
}

// Resolve selects the effective server for a command.
func Resolve(repo *config.Repository, bound *config.Server, opts Options) (*Resolved, error) {
	if opts.Command != "" && opts.URL != "" {
		return nil, exitcode.New(exitcode.Usage, "--command and --url are mutually exclusive")
	}
	if bound != nil && (opts.ExplicitName != "" || opts.Command != "" || opts.URL != "") {
		return nil, exitcode.New(exitcode.Usage, "bound exposed commands do not accept an explicit server, --command, or --url")
	}

	if bound != nil {
		server := cloneServer(bound)
		if err := applyOverrides(server, opts); err != nil {
			return nil, err
		}
		return &Resolved{DisplayName: server.Name, Server: server}, nil
	}

	if opts.Command != "" || opts.URL != "" {
		server := &config.Server{
			Name:    "(direct)",
			Source:  config.SourceEphemeral,
			Command: strings.TrimSpace(opts.Command),
			URL:     strings.TrimSpace(opts.URL),
		}
		if err := applyOverrides(server, opts); err != nil {
			return nil, err
		}
		display := server.Command
		if server.URL != "" {
			display = server.URL
		}
		return &Resolved{DisplayName: display, Server: server}, nil
	}

	if opts.ExplicitName == "" {
		return nil, exitcode.New(exitcode.Usage, "a registered server name or one of --command/--url is required")
	}

	server, err := repo.ResolveServer(opts.ExplicitName)
	if err != nil {
		return nil, exitcode.WithHint(exitcode.Wrapf(exitcode.Config, err, "server %q not found", opts.ExplicitName), "run `mcp2cli ls`")
	}
	if err := applyOverrides(server, opts); err != nil {
		return nil, err
	}
	return &Resolved{DisplayName: server.Name, Server: server}, nil
}

func applyOverrides(server *config.Server, opts Options) error {
	if opts.CWD != "" {
		server.CWD = opts.CWD
	}
	if opts.Auth != "" {
		server.Auth = opts.Auth
	}
	if opts.BearerEnv != "" {
		server.BearerEnv = opts.BearerEnv
	}
	if opts.OAuthAuthorizeURL != "" {
		server.OAuthAuthorizeURL = opts.OAuthAuthorizeURL
	}
	if opts.OAuthTokenURL != "" {
		server.OAuthTokenURL = opts.OAuthTokenURL
	}
	if opts.OAuthClientID != "" {
		server.OAuthClientID = opts.OAuthClientID
	}
	if len(opts.OAuthScopes) > 0 {
		server.OAuthScopes = append([]string(nil), opts.OAuthScopes...)
	}
	if len(opts.Env) > 0 {
		if server.Env == nil {
			server.Env = map[string]string{}
		}
		for _, entry := range opts.Env {
			key, value, ok := strings.Cut(entry, "=")
			if !ok || key == "" {
				return exitcode.Newf(exitcode.Usage, "invalid --env value %q, expected KEY=VALUE", entry)
			}
			server.Env[key] = value
		}
	}
	if len(opts.Headers) > 0 {
		if server.Headers == nil {
			server.Headers = map[string]string{}
		}
		for _, entry := range opts.Headers {
			key, value, ok := strings.Cut(entry, ":")
			if !ok || strings.TrimSpace(key) == "" {
				return exitcode.Newf(exitcode.Usage, "invalid --header value %q, expected Key: Value", entry)
			}
			server.Headers[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
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
	if server.OAuthScopes != nil {
		clone.OAuthScopes = append([]string(nil), server.OAuthScopes...)
	}
	if server.ExposeAs != nil {
		clone.ExposeAs = append([]string(nil), server.ExposeAs...)
	}
	return &clone
}
