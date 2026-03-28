package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

// ConfigSource identifies where a server configuration came from.
type ConfigSource string

const (
	SourceGlobal    ConfigSource = "global"
	SourceLocal     ConfigSource = "local"
	SourceEphemeral ConfigSource = "ephemeral"
)

var commandNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// Paths contains the filesystem paths used by mcp2cli.
type Paths struct {
	GlobalConfig string
	LocalConfig  string
	ExposeBinDir string
	TokenDir     string
}

// File is the on-disk configuration format.
type File struct {
	Version  int                `yaml:"version,omitempty"`
	Defaults DefaultsConfig     `yaml:"defaults,omitempty"`
	Servers  map[string]*Server `yaml:"servers,omitempty"`
}

// DefaultsConfig stores CLI defaults.
type DefaultsConfig struct {
	Output  string `yaml:"output,omitempty"`
	Color   string `yaml:"color,omitempty"`
	Timeout string `yaml:"timeout,omitempty"`
}

// Server contains a registered server definition.
type Server struct {
	Name              string            `yaml:"-"`
	Source            ConfigSource      `yaml:"-"`
	Transport         string            `yaml:"transport,omitempty"`
	Command           string            `yaml:"command,omitempty"`
	URL               string            `yaml:"url,omitempty"`
	CWD               string            `yaml:"cwd,omitempty"`
	Env               map[string]string `yaml:"env,omitempty"`
	Roots             []string          `yaml:"roots,omitempty"`
	Headers           map[string]string `yaml:"headers,omitempty"`
	Auth              string            `yaml:"auth,omitempty"`
	BearerEnv         string            `yaml:"bearer_env,omitempty"`
	OAuthAuthorizeURL string            `yaml:"oauth_authorize_url,omitempty"`
	OAuthTokenURL     string            `yaml:"oauth_token_url,omitempty"`
	OAuthClientID     string            `yaml:"oauth_client_id,omitempty"`
	OAuthScopes       []string          `yaml:"oauth_scopes,omitempty"`
	ExposeAs          []string          `yaml:"expose,omitempty"`
	Timeout           string            `yaml:"timeout,omitempty"`
}

// Repository provides read/write access to config files.
type Repository struct {
	Paths Paths
}

// DefaultPaths resolves the standard config and data locations.
func DefaultPaths(cwd string) (Paths, error) {
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return Paths{}, fmt.Errorf("get working directory: %w", err)
		}
	}

	return Paths{
		GlobalConfig: filepath.Join(xdg.ConfigHome, "mcp2cli", "config.yaml"),
		LocalConfig:  filepath.Join(cwd, ".mcp2cli.yaml"),
		ExposeBinDir: filepath.Join(xdg.DataHome, "mcp2cli", "bin"),
		TokenDir:     filepath.Join(xdg.DataHome, "mcp2cli", "tokens"),
	}, nil
}

// NewRepository constructs a repository using standard paths.
func NewRepository(cwd string) (*Repository, error) {
	paths, err := DefaultPaths(cwd)
	if err != nil {
		return nil, err
	}
	return &Repository{Paths: paths}, nil
}

// NewRepositoryWithPaths constructs a repository with explicit paths. It is
// mainly useful for tests.
func NewRepositoryWithPaths(paths Paths) *Repository {
	return &Repository{Paths: paths}
}

// LoadMerged loads global and local config and applies local-over-global server
// precedence.
func (r *Repository) LoadMerged() (*File, error) {
	global, err := r.LoadGlobal()
	if err != nil {
		return nil, err
	}
	local, err := r.LoadLocal()
	if err != nil {
		return nil, err
	}

	merged := newFile()
	merged.Version = max(global.Version, local.Version)
	merged.Defaults = mergeDefaults(global.Defaults, local.Defaults)

	for name, server := range global.Servers {
		merged.Servers[name] = cloneServer(server)
		merged.Servers[name].Name = name
		merged.Servers[name].Source = SourceGlobal
	}
	for name, server := range local.Servers {
		merged.Servers[name] = cloneServer(server)
		merged.Servers[name].Name = name
		merged.Servers[name].Source = SourceLocal
	}

	return merged, nil
}

// LoadGlobal loads the global config file.
func (r *Repository) LoadGlobal() (*File, error) {
	return loadFile(r.Paths.GlobalConfig)
}

// LoadLocal loads the local config file.
func (r *Repository) LoadLocal() (*File, error) {
	return loadFile(r.Paths.LocalConfig)
}

// SaveGlobal writes the global config file atomically.
func (r *Repository) SaveGlobal(file *File) error {
	return saveFile(r.Paths.GlobalConfig, file)
}

// SaveLocal writes the local config file atomically.
func (r *Repository) SaveLocal(file *File) error {
	return saveFile(r.Paths.LocalConfig, file)
}

// UpsertServer creates or replaces a server in the selected scope.
func (r *Repository) UpsertServer(scope ConfigSource, name string, server *Server) error {
	normalizedName, err := NormalizeCommandName(name)
	if err != nil {
		return fmt.Errorf("normalize server name: %w", err)
	}

	file, err := r.loadForScope(scope)
	if err != nil {
		return err
	}

	serverCopy := cloneServer(server)
	serverCopy.Name = ""
	serverCopy.Source = ""
	serverCopy.ExposeAs, err = normalizeExposeNames(serverCopy.ExposeAs)
	if err != nil {
		return err
	}

	file.Servers[normalizedName] = serverCopy
	return r.saveForScope(scope, file)
}

// RemoveServer removes a server from the selected scope.
func (r *Repository) RemoveServer(scope ConfigSource, name string) error {
	normalizedName, err := NormalizeCommandName(name)
	if err != nil {
		return fmt.Errorf("normalize server name: %w", err)
	}

	file, err := r.loadForScope(scope)
	if err != nil {
		return err
	}

	if _, ok := file.Servers[normalizedName]; !ok {
		return fmt.Errorf("server %q not found in %s config", normalizedName, scope)
	}

	delete(file.Servers, normalizedName)
	return r.saveForScope(scope, file)
}

// ResolveServer finds a server by registered name in merged config.
func (r *Repository) ResolveServer(name string) (*Server, error) {
	normalizedName, err := NormalizeCommandName(name)
	if err != nil {
		return nil, fmt.Errorf("normalize server name: %w", err)
	}

	merged, err := r.LoadMerged()
	if err != nil {
		return nil, err
	}

	server, ok := merged.Servers[normalizedName]
	if !ok {
		return nil, fmt.Errorf("server %q not found", normalizedName)
	}

	return cloneServer(server), nil
}

// ResolveExposedCommand finds a server by exposed command name.
func (r *Repository) ResolveExposedCommand(name string) (*Server, error) {
	normalizedName, err := NormalizeCommandName(name)
	if err != nil {
		return nil, fmt.Errorf("normalize exposed command name: %w", err)
	}

	merged, err := r.LoadMerged()
	if err != nil {
		return nil, err
	}

	var match *Server
	for _, server := range merged.Servers {
		for _, exposedName := range server.ExposeAs {
			if exposedName == normalizedName {
				if match != nil && match.Name != server.Name {
					return nil, fmt.Errorf("exposed command %q is assigned to multiple servers", normalizedName)
				}
				match = cloneServer(server)
			}
		}
	}

	if match == nil {
		return nil, fmt.Errorf("exposed command %q is not registered", normalizedName)
	}

	return match, nil
}

// AddExpose adds an exposed command name to a server in the given scope.
func (r *Repository) AddExpose(scope ConfigSource, serverName, exposedName string) error {
	normalizedServerName, err := NormalizeCommandName(serverName)
	if err != nil {
		return fmt.Errorf("normalize server name: %w", err)
	}
	normalizedExposeName, err := NormalizeCommandName(exposedName)
	if err != nil {
		return fmt.Errorf("normalize exposed command name: %w", err)
	}

	merged, err := r.LoadMerged()
	if err != nil {
		return err
	}
	for _, server := range merged.Servers {
		for _, existing := range server.ExposeAs {
			if existing == normalizedExposeName && server.Name != normalizedServerName {
				return fmt.Errorf("exposed command %q is already used by server %q", normalizedExposeName, server.Name)
			}
		}
	}

	file, err := r.loadForScope(scope)
	if err != nil {
		return err
	}

	server, ok := file.Servers[normalizedServerName]
	if !ok {
		return fmt.Errorf("server %q not found in %s config", normalizedServerName, scope)
	}

	for _, existing := range server.ExposeAs {
		if existing == normalizedExposeName {
			return nil
		}
	}

	server.ExposeAs = append(server.ExposeAs, normalizedExposeName)
	sort.Strings(server.ExposeAs)
	return r.saveForScope(scope, file)
}

// RemoveExpose removes an exposed command name from a server in the given scope.
func (r *Repository) RemoveExpose(scope ConfigSource, serverName, exposedName string) error {
	normalizedServerName, err := NormalizeCommandName(serverName)
	if err != nil {
		return fmt.Errorf("normalize server name: %w", err)
	}
	normalizedExposeName, err := NormalizeCommandName(exposedName)
	if err != nil {
		return fmt.Errorf("normalize exposed command name: %w", err)
	}

	file, err := r.loadForScope(scope)
	if err != nil {
		return err
	}

	server, ok := file.Servers[normalizedServerName]
	if !ok {
		return fmt.Errorf("server %q not found in %s config", normalizedServerName, scope)
	}

	filtered := server.ExposeAs[:0]
	removed := false
	for _, existing := range server.ExposeAs {
		if existing == normalizedExposeName {
			removed = true
			continue
		}
		filtered = append(filtered, existing)
	}
	if !removed {
		return fmt.Errorf("exposed command %q is not assigned to server %q", normalizedExposeName, normalizedServerName)
	}
	server.ExposeAs = filtered
	return r.saveForScope(scope, file)
}

// ListServers returns all merged servers sorted by name.
func (r *Repository) ListServers() ([]*Server, error) {
	merged, err := r.LoadMerged()
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(merged.Servers))
	for name := range merged.Servers {
		names = append(names, name)
	}
	sort.Strings(names)

	servers := make([]*Server, 0, len(names))
	for _, name := range names {
		servers = append(servers, cloneServer(merged.Servers[name]))
	}
	return servers, nil
}

// DefaultExposeName returns the default exposed command name for a server.
func DefaultExposeName(serverName string) (string, error) {
	normalizedName, err := NormalizeCommandName(serverName)
	if err != nil {
		return "", err
	}
	return "mcp-" + normalizedName, nil
}

// NormalizeCommandName normalizes a command or alias to kebab-case.
func NormalizeCommandName(name string) (string, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return "", errors.New("name cannot be empty")
	}
	if strings.ContainsRune(name, os.PathSeparator) || strings.Contains(name, string(filepath.Separator)) {
		return "", errors.New("name cannot contain path separators")
	}

	replacer := strings.NewReplacer("_", "-", " ", "-")
	name = replacer.Replace(name)
	name = strings.Trim(name, "-")
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}

	if !commandNamePattern.MatchString(name) {
		return "", fmt.Errorf("invalid name %q: use lowercase letters, numbers, and dashes only", name)
	}
	if name == "mcp2cli" {
		return "", errors.New("name \"mcp2cli\" is reserved")
	}

	return name, nil
}

func loadFile(path string) (*File, error) {
	file := newFile()

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return file, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	if len(data) == 0 {
		return file, nil
	}
	if err := yaml.Unmarshal(data, file); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if file.Version == 0 {
		file.Version = 1
	}
	if file.Servers == nil {
		file.Servers = map[string]*Server{}
	}
	for name, server := range file.Servers {
		normalizedName, err := NormalizeCommandName(name)
		if err != nil {
			return nil, fmt.Errorf("invalid server name %q in %s: %w", name, path, err)
		}
		if normalizedName != name {
			delete(file.Servers, name)
			file.Servers[normalizedName] = server
		}
		server.ExposeAs, err = normalizeExposeNames(server.ExposeAs)
		if err != nil {
			return nil, fmt.Errorf("invalid expose list for server %q in %s: %w", normalizedName, path, err)
		}
	}
	return file, nil
}

func saveFile(path string, file *File) error {
	if file == nil {
		return errors.New("config file cannot be nil")
	}
	if file.Version == 0 {
		file.Version = 1
	}
	if file.Servers == nil {
		file.Servers = map[string]*Server{}
	}

	data, err := yaml.Marshal(file)
	if err != nil {
		return fmt.Errorf("marshal config %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config directory for %s: %w", path, err)
	}

	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o644); err != nil {
		return fmt.Errorf("write temp config %s: %w", tempPath, err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace config %s: %w", path, err)
	}
	return nil
}

func (r *Repository) loadForScope(scope ConfigSource) (*File, error) {
	switch scope {
	case SourceGlobal:
		return r.LoadGlobal()
	case SourceLocal:
		return r.LoadLocal()
	default:
		return nil, fmt.Errorf("unsupported config scope %q", scope)
	}
}

func (r *Repository) saveForScope(scope ConfigSource, file *File) error {
	switch scope {
	case SourceGlobal:
		return r.SaveGlobal(file)
	case SourceLocal:
		return r.SaveLocal(file)
	default:
		return fmt.Errorf("unsupported config scope %q", scope)
	}
}

func mergeDefaults(global, local DefaultsConfig) DefaultsConfig {
	merged := global
	if local.Output != "" {
		merged.Output = local.Output
	}
	if local.Color != "" {
		merged.Color = local.Color
	}
	if local.Timeout != "" {
		merged.Timeout = local.Timeout
	}
	return merged
}

func cloneServer(server *Server) *Server {
	if server == nil {
		return &Server{}
	}

	clone := *server
	if server.Env != nil {
		clone.Env = make(map[string]string, len(server.Env))
		for key, value := range server.Env {
			clone.Env[key] = value
		}
	}
	if server.Headers != nil {
		clone.Headers = make(map[string]string, len(server.Headers))
		for key, value := range server.Headers {
			clone.Headers[key] = value
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

func newFile() *File {
	return &File{
		Version: 1,
		Servers: map[string]*Server{},
	}
}

func normalizeExposeNames(names []string) ([]string, error) {
	if len(names) == 0 {
		return nil, nil
	}

	unique := map[string]struct{}{}
	normalized := make([]string, 0, len(names))
	for _, name := range names {
		normalizedName, err := NormalizeCommandName(name)
		if err != nil {
			return nil, err
		}
		if _, ok := unique[normalizedName]; ok {
			continue
		}
		unique[normalizedName] = struct{}{}
		normalized = append(normalized, normalizedName)
	}
	sort.Strings(normalized)
	return normalized, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
