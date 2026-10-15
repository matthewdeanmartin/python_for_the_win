package proc

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"
)

type Options struct {
	Dir     string
	Timeout time.Duration
	Env     []string
}
type Runner interface {
	Run(context.Context, string, []string, Options) (string, string, int, error)
}
type Real struct{}

func (Real) Run(ctx context.Context, exe string, args []string, o Options) (string, string, int, error) {
	if !filepath.IsAbs(exe) {
		return "", "", -1, fmt.Errorf("child executable must be an absolute path")
	}
	if o.Timeout == 0 {
		o.Timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, o.Timeout)
	defer cancel()
	c := exec.CommandContext(ctx, exe, args...)
	c.Dir = o.Dir
	c.Env = o.Env
	c.WaitDelay = 2 * time.Second
	var out, errout bytes.Buffer
	c.Stdout = &out
	c.Stderr = &errout
	e := c.Run()
	code := 0
	if c.ProcessState != nil {
		code = c.ProcessState.ExitCode()
	}
	if ctx.Err() != nil {
		e = ctx.Err()
	}
	return out.String(), errout.String(), code, e
}
