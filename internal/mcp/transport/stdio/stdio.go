package stdio

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/kballard/go-shellquote"
	"github.com/maximerivest/mcp2cli/internal/exitcode"
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
		return nil, exitcode.Wrapf(exitcode.Usage, err, "parse command %q", command)
	}
	if len(argv) == 0 {
		return nil, exitcode.New(exitcode.Config, "command cannot be empty")
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
		return nil, exitcode.Wrap(exitcode.Internal, err, "open stdin pipe")
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, exitcode.Wrap(exitcode.Internal, err, "open stdout pipe")
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
		return nil, exitcode.Wrapf(exitcode.Transport, err, "start command %q", argv[0])
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
				closeErr = exitcode.Wrapf(exitcode.Transport, err, "wait for subprocess%s", t.stderrSuffix())
			}
			return
		case <-time.After(300 * time.Millisecond):
		}

		t.cancel()
		select {
		case err := <-t.done:
			if err != nil && !isProcessExitAfterCancel(err) {
				closeErr = exitcode.Wrapf(exitcode.Transport, err, "wait for subprocess after cancel%s", t.stderrSuffix())
			}
		case <-time.After(2 * time.Second):
			_ = t.cmd.Process.Kill()
			<-t.done
		}
	})
	return closeErr
}

// Stderr returns any captured stderr output from the subprocess.
func (t *Transport) Stderr() string {
	return strings.TrimSpace(t.stderr.String())
}

// Exited reports whether the subprocess has already exited and its exit code.
// Returns (false, 0) if the process is still running.
func (t *Transport) Exited() (bool, int) {
	select {
	case err := <-t.done:
		// Put the result back so Close() still sees it
		t.done <- err
		if err == nil {
			return true, 0
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return true, exitErr.ExitCode()
		}
		return true, 1
	default:
		return false, 0
	}
}

func (t *Transport) stderrSuffix() string {
	stderr := t.Stderr()
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
