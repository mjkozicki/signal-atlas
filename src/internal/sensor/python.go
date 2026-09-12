package sensor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// PythonRunner executes only the embedded provider program, with fixed argument names.
// The interpreter path is configured at process startup, never from an HTTP request.
type PythonRunner struct {
	Executable string
	Script     string
}

func PythonPath(explicit string) string {
	if explicit != "" {
		return explicit
	}
	rel := filepath.Join(".venv", "bin", "python3")
	if runtime.GOOS == "windows" {
		rel = filepath.Join(".venv", "Scripts", "python.exe")
	}
	if exe, e := os.Executable(); e == nil {
		p := filepath.Join(filepath.Dir(exe), "..", rel)
		if _, e = os.Stat(p); e == nil {
			return p
		}
	}
	if p, e := filepath.Abs(rel); e == nil {
		if _, e = os.Stat(p); e == nil {
			return p
		}
	}
	return "python3"
}

type limitedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.limit {
		return 0, fmt.Errorf("provider output exceeds size limit")
	}
	return b.Buffer.Write(p)
}
func (p PythonRunner) Run(ctx context.Context, action string, o Options, out any) error {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(o.Duration+10)*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, p.Executable, "-c", p.Script, "--action", action, "--duration", fmt.Sprint(o.Duration), "--adapter", o.Adapter, "--reader", o.Reader)
	command.Env = append(os.Environ(), "PYTHONDONTWRITEBYTECODE=1", "PYTHONUNBUFFERED=1")
	stdout, stderr := &limitedBuffer{limit: 4 << 20}, &limitedBuffer{limit: 32 << 10}
	command.Stdout = stdout
	command.Stderr = stderr
	command.WaitDelay = time.Second
	if e := command.Run(); e != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("provider stopped: %w", ctx.Err())
		}
		return fmt.Errorf("native provider failed: %w: %s", e, stderr.String())
	}
	d := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	if e := d.Decode(out); e != nil {
		return fmt.Errorf("invalid native provider response: %w", e)
	}
	if e := d.Decode(new(any)); e != io.EOF {
		return fmt.Errorf("native provider response contains trailing data")
	}
	return nil
}
