package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	mcpjsonrpc "github.com/maximerivest/mcp2cli/internal/mcp/jsonrpc"
	"github.com/maximerivest/mcp2cli/internal/mcp/transport/stdio"
	"github.com/maximerivest/mcp2cli/internal/mcp/types"
)

// Paths returns the socket and pid file paths for a server.
func Paths(dataDir, serverName string) (socketPath, pidPath string) {
	dir := filepath.Join(dataDir, "mcp2cli", "daemons")
	return filepath.Join(dir, serverName+".sock"), filepath.Join(dir, serverName+".pid")
}

// IsRunning checks if a daemon is running for the given server.
func IsRunning(dataDir, serverName string) bool {
	socketPath, pidPath := Paths(dataDir, serverName)
	pid, err := readPID(pidPath)
	if err != nil || pid == 0 {
		return false
	}
	// Check if process exists
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		cleanup(socketPath, pidPath)
		return false
	}
	// Check socket is connectable
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// SocketURL returns the HTTP URL for a running daemon.
func SocketURL(dataDir, serverName string) string {
	return "http://unix/" + serverName
}

// DialSocket returns an HTTP client that connects to the daemon's Unix socket.
func DialSocket(dataDir, serverName string) *http.Client {
	socketPath, _ := Paths(dataDir, serverName)
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}
}

// Run starts the proxy daemon in the foreground (called by the detached child).
func Run(ctx context.Context, command, dataDir, serverName string) error {
	socketPath, pidPath := Paths(dataDir, serverName)

	if err := os.MkdirAll(filepath.Dir(socketPath), 0o755); err != nil {
		return fmt.Errorf("create daemon directory: %w", err)
	}
	cleanup(socketPath, pidPath)

	// Start MCP server
	transport, err := stdio.Start(ctx, command, "", nil)
	if err != nil {
		return fmt.Errorf("start server: %w", err)
	}
	defer transport.Close()

	// JSON-RPC client on the server's stdio
	rpc := mcpjsonrpc.NewClient(transport.Reader(), transport.Writer(), nil, nil)

	// Initialize handshake
	var initResult types.InitializeResult
	if err := rpc.Call(ctx, "initialize", types.InitializeParams{
		ProtocolVersion: "2024-11-05",
		Capabilities:    map[string]any{},
		ClientInfo:      types.Implementation{Name: "mcp2cli-daemon", Version: "dev"},
	}, &initResult); err != nil {
		return fmt.Errorf("initialize: %w", err)
	}
	_ = rpc.Notify(ctx, "notifications/initialized", map[string]any{})

	// Listen on Unix socket
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("listen on socket: %w", err)
	}
	defer listener.Close()
	os.Chmod(socketPath, 0o600)

	// Write PID file
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		return fmt.Errorf("write pid file: %w", err)
	}
	defer cleanup(socketPath, pidPath)

	// Handle shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	// HTTP proxy: forward JSON-RPC to the running server
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var request struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(body, &request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if request.Method == "notifications/initialized" || request.Method == "initialize" {
			// Already initialized — return cached result
			w.Header().Set("Content-Type", "application/json")
			result, _ := json.Marshal(map[string]any{
				"jsonrpc": "2.0",
				"id":      json.RawMessage(`1`),
				"result":  initResult,
			})
			w.Write(result)
			return
		}
		var result json.RawMessage
		if err := rpc.Call(r.Context(), request.Method, request.Params, &result); err != nil {
			w.Header().Set("Content-Type", "application/json")
			errPayload, _ := json.Marshal(map[string]any{
				"jsonrpc": "2.0",
				"id":      json.RawMessage(`1`),
				"error":   map[string]any{"code": -32000, "message": err.Error()},
			})
			w.Write(errPayload)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		payload, _ := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"id":      json.RawMessage(`1`),
			"result":  result,
		})
		w.Write(payload)
	})}

	go func() {
		<-sigCh
		server.Close()
	}()

	return server.Serve(listener)
}

// SharedPaths returns the URL file and pid file paths for a shared (HTTP) server.
func SharedPaths(dataDir, serverName string) (urlPath, pidPath string) {
	dir := filepath.Join(dataDir, "mcp2cli", "daemons")
	return filepath.Join(dir, serverName+".url"), filepath.Join(dir, serverName+".pid")
}

// IsSharedRunning checks if a shared HTTP daemon is running and reachable.
func IsSharedRunning(dataDir, serverName string) bool {
	url, err := SharedURL(dataDir, serverName)
	if err != nil || url == "" {
		return false
	}
	_, pidPath := SharedPaths(dataDir, serverName)
	pid, err := readPID(pidPath)
	if err != nil || pid == 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return false
	}
	client := &http.Client{Timeout: 250 * time.Millisecond}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return true
}

// SharedURL reads the URL of a running shared server.
func SharedURL(dataDir, serverName string) (string, error) {
	urlPath, _ := SharedPaths(dataDir, serverName)
	data, err := os.ReadFile(urlPath)
	if err != nil {
		return "", fmt.Errorf("read shared URL: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// findFreePort finds an available TCP port.
func findFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port, nil
}

// RunShared starts the server command with --http --port and waits for it.
// The server itself handles MCP HTTP — we just manage the process lifecycle.
func RunShared(ctx context.Context, command, dataDir, serverName string) error {
	urlPath, pidPath := SharedPaths(dataDir, serverName)

	if err := os.MkdirAll(filepath.Dir(urlPath), 0o755); err != nil {
		return fmt.Errorf("create daemon directory: %w", err)
	}

	// Clean up stale files
	os.Remove(urlPath)
	os.Remove(pidPath)

	// Find a free port
	port, err := findFreePort()
	if err != nil {
		return fmt.Errorf("find free port: %w", err)
	}

	// Append --http --port N to the command
	fullCommand := fmt.Sprintf("%s --http --port %d", command, port)

	// Start the server as a subprocess (not stdio — it serves HTTP itself)
	devnull, _ := os.Open(os.DevNull)
	cmd := exec.CommandContext(ctx, "sh", "-c", fullCommand)
	cmd.Stdout = devnull
	cmd.Stderr = devnull
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/mcp", port)

	// Wait for the HTTP server to be ready.
	// Try connecting to /mcp — any response (even 405) means the server is up.
	httpClient := &http.Client{Timeout: 2 * time.Second}
	ready := false
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		checkURL := fmt.Sprintf("http://127.0.0.1:%d/mcp", port)
		resp, err := httpClient.Get(checkURL)
		if err == nil {
			resp.Body.Close()
			ready = true
			break
		}
	}

	if !ready {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		return fmt.Errorf("server did not become ready on port %d", port)
	}

	// Only publish the URL/PID after the server is actually reachable.
	serverPid := cmd.Process.Pid
	if err := os.WriteFile(urlPath, []byte(url+"\n"), 0o644); err != nil {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		return fmt.Errorf("write url file: %w", err)
	}
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(serverPid)), 0o644); err != nil {
		os.Remove(urlPath)
		_ = cmd.Process.Signal(syscall.SIGTERM)
		return fmt.Errorf("write pid file: %w", err)
	}
	defer func() {
		os.Remove(urlPath)
		os.Remove(pidPath)
	}()

	// Wait for signal — then kill the server
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh

	_ = cmd.Process.Signal(syscall.SIGTERM)
	_ = cmd.Wait()
	return nil
}

// StopShared stops a shared HTTP daemon.
func StopShared(dataDir, serverName string) error {
	urlPath, pidPath := SharedPaths(dataDir, serverName)
	pid, err := readPID(pidPath)
	if err != nil || pid == 0 {
		os.Remove(urlPath)
		os.Remove(pidPath)
		return fmt.Errorf("server %q is not running", serverName)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(urlPath)
		os.Remove(pidPath)
		return nil
	}
	_ = proc.Signal(syscall.SIGTERM)
	os.Remove(urlPath)
	os.Remove(pidPath)
	return nil
}

// Stop sends SIGTERM to the daemon and cleans up.
func Stop(dataDir, serverName string) error {
	socketPath, pidPath := Paths(dataDir, serverName)
	pid, err := readPID(pidPath)
	if err != nil || pid == 0 {
		cleanup(socketPath, pidPath)
		return fmt.Errorf("server %q is not running", serverName)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		cleanup(socketPath, pidPath)
		return nil
	}
	_ = proc.Signal(syscall.SIGTERM)
	cleanup(socketPath, pidPath)
	return nil
}

func readPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(data))
}

func cleanup(socketPath, pidPath string) {
	os.Remove(socketPath)
	os.Remove(pidPath)
}
