package stdio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/kballard/go-shellquote"
)

// Transport manages a stdio-based MCP server subprocess.
type Transport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	cancel context.CancelFunc
	done   chan error
	stderr bytes.Buffer
	once   sync.Once
}

// Start launches a stdio MCP server using the provided command string.
func Start(parent context.Context, command, cwd string, env map[string]string) (*Transport, error) {
	argv, err := shellquote.Split(command)
	if err != nil {
		return nil, fmt.Errorf("parse command %q: %w", command, err)
	}
	if len(argv) == 0 {
		return nil, fmt.Errorf("command cannot be empty")
	}

	ctx, cancel := context.WithCancel(parent)
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	if len(env) > 0 {
		mergedEnv := os.Environ()
		for key, value := range env {
			mergedEnv = append(mergedEnv, key+"="+value)
		}
		cmd.Env = mergedEnv
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("open stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("open stdout pipe: %w", err)
	}
	transport := &Transport{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		cancel: cancel,
		done:   make(chan error, 1),
	}
	cmd.Stderr = &transport.stderr

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start command %q: %w", argv[0], err)
	}
	go func() {
		transport.done <- cmd.Wait()
	}()

	return transport, nil
}

// Reader returns the subprocess stdout.
func (t *Transport) Reader() io.Reader { return t.stdout }

// Writer returns the subprocess stdin.
func (t *Transport) Writer() io.Writer { return t.stdin }

// Close gracefully shuts down the subprocess and reaps it.
func (t *Transport) Close() error {
	var closeErr error
	if t == nil {
		return nil
	}

	t.once.Do(func() {
		if t.stdin != nil {
			_ = t.stdin.Close()
		}

		select {
		case err := <-t.done:
			if err != nil && !isProcessExitAfterCancel(err) {
				closeErr = fmt.Errorf("wait for subprocess: %w%s", err, t.stderrSuffix())
			}
			return
		case <-time.After(300 * time.Millisecond):
		}

		t.cancel()
		select {
		case err := <-t.done:
			if err != nil && !isProcessExitAfterCancel(err) {
				closeErr = fmt.Errorf("wait for subprocess after cancel: %w%s", err, t.stderrSuffix())
			}
		case <-time.After(2 * time.Second):
			_ = t.cmd.Process.Kill()
			<-t.done
		}
	})
	return closeErr
}

func (t *Transport) stderrSuffix() string {
	stderr := strings.TrimSpace(t.stderr.String())
	if stderr == "" {
		return ""
	}
	return ": stderr: " + stderr
}

func isProcessExitAfterCancel(err error) bool {
	if err == nil {
		return true
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode() != 0
	}
	return false
}
