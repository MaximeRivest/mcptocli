package cli

import (
	"fmt"
	"os"

	"github.com/maximerivest/mcp2cli/internal/app"
	"github.com/maximerivest/mcp2cli/internal/auth"
	"github.com/maximerivest/mcp2cli/internal/cache"
	"github.com/maximerivest/mcp2cli/internal/config"
	"github.com/maximerivest/mcp2cli/internal/exitcode"
)

// Options configures the CLI.
type Options struct {
	Version    string
	Commit     string
	BuildDate  string
	Invocation app.Invocation
}

// State carries shared runtime state for commands.
type State struct {
	Options Options
	cwd     string
}

func newState(opts Options) (*State, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}
	return &State{Options: opts, cwd: cwd}, nil
}

func (s *State) Repo() (*config.Repository, error) {
	return config.NewRepository(s.cwd)
}

func (s *State) TokenStore() (*auth.Store, error) {
	repo, err := s.Repo()
	if err != nil {
		return nil, err
	}
	return auth.NewStore(repo.Paths), nil
}

func (s *State) MetadataStore() (*cache.Store, error) {
	repo, err := s.Repo()
	if err != nil {
		return nil, err
	}
	return cache.NewStore(repo.Paths), nil
}

func (s *State) BoundServer() (*config.Server, error) {
	if !s.Options.Invocation.IsExposedCommand() {
		return nil, nil
	}

	repo, err := s.Repo()
	if err != nil {
		return nil, err
	}

	name := s.Options.Invocation.ExposedCommandName

	// Implicit bind: "mcp2cli weather ..." — resolve by server name directly
	if s.Options.Invocation.ImplicitBind {
		server, err := repo.ResolveServer(name)
		if err != nil {
			return nil, exitcode.WithHint(exitcode.Wrapf(exitcode.Config, err, "server %q not found", name), "run `mcp2cli ls`")
		}
		return server, nil
	}

	// Exposed command: "mcp-weather ..." or "wea ..." — resolve by exposed name
	server, err := repo.ResolveExposedCommand(name)
	if err != nil {
		return nil, exitcode.WithHint(exitcode.Wrapf(exitcode.Config, err, "exposed command %q is not registered", name), "run `mcp2cli ls`")
	}
	return server, nil
}
