package sbpm

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// Executor runs external commands.
// Implementations must be safe for concurrent use.
//
type Executor interface {
	Run(ctx context.Context, name string, args ...string) (stdout, stderr []byte, err error)
}

// DefaultExecutor uses exec.CommandContext and captures stdout/stderr.
// It does not invoke a shell; args are passed directly.
//
type DefaultExecutor struct{}

func (d *DefaultExecutor) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err != nil {
		return outBuf.Bytes(), errBuf.Bytes(), fmt.Errorf("exec %s %v: %w", name, args, err)
	}
	return outBuf.Bytes(), errBuf.Bytes(), nil
}

// FakeExecutor records invocations and returns canned responses. Useful in tests.
// Not concurrency-safe; use per-test.
//
type FakeExecutor struct {
	Calls []ExecCall
	// Map key: name + "\x00" + strings.Join(args, "\x00")
	Responses map[string]ExecResponse
}

type ExecCall struct {
	Name string
	Args []string
}

type ExecResponse struct {
	Stdout []byte
	Stderr []byte
	Err    error
}

func (f *FakeExecutor) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	f.Calls = append(f.Calls, ExecCall{Name: name, Args: append([]string(nil), args...)})
	key := name + "\x00" + join0(args)
	if f.Responses != nil {
		if r, ok := f.Responses[key]; ok {
			return r.Stdout, r.Stderr, r.Err
		}
	}
	return nil, nil, nil
}

func join0(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	b := ss[0]
	for i := 1; i < len(ss); i++ {
		b += "\x00" + ss[i]
	}
	return b
}
