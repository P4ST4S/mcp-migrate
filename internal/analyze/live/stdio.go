package live

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/P4ST4S/mcp-migrate/internal/spec"
)

const (
	defaultRPCTimeout = 2 * time.Second
	maxStderrBytes    = 32 * 1024
)

type stdioProbe struct {
	name     string
	method   string
	params   map[string]any
	readOnly bool
	mutating bool
}

func ProbeSTDIO(opts Options) (STDIOTrace, error) {
	if opts.ServerCommand == "" {
		return STDIOTrace{}, fmt.Errorf("--server-command is required for stdio analysis")
	}
	args, err := splitCommandLine(opts.ServerCommand)
	if err != nil {
		return STDIOTrace{}, err
	}
	if len(args) == 0 {
		return STDIOTrace{}, fmt.Errorf("--server-command is empty")
	}

	rpcTimeout := opts.Timeout
	if rpcTimeout == 0 {
		rpcTimeout = defaultRPCTimeout
	}
	specTarget := opts.SpecTarget
	if specTarget == "" {
		specTarget = spec.TargetVersion
	}

	session, err := startSTDIOSession(args)
	if err != nil {
		return STDIOTrace{}, err
	}
	defer session.stop()

	trace := STDIOTrace{
		Command:             sanitizeCommand(args),
		AllowMutatingProbes: opts.AllowMutatingProbes,
	}

	discover := session.runProbe(stdioProbe{
		name:     "discover",
		method:   "server/discover",
		params:   paramsWithMeta(specTarget),
		readOnly: true,
	}, rpcTimeout)
	trace.Observations = append(trace.Observations, discover)

	if !discover.Accepted() || !hasDiscoverShape(discover.Result) {
		// Unlike HTTP, stdio analysis launches an isolated process owned by this
		// analyzer and tears it down after probing. Sending initialize here is a
		// compatibility probe for process-scoped legacy servers, not a mutation
		// against a shared remote server.
		trace.SentLegacyInitialize = true
		trace.Observations = append(trace.Observations, session.runProbe(stdioProbe{
			name:     "initialize-legacy",
			method:   "initialize",
			params:   legacyInitializeParams(specTarget),
			readOnly: true,
		}, rpcTimeout))
	}

	for _, probe := range defaultSTDIOProbes(specTarget) {
		if probe.mutating && !opts.AllowMutatingProbes {
			continue
		}
		trace.Observations = append(trace.Observations, session.runProbe(probe, rpcTimeout))
	}

	trace.StderrBytes = session.stderr.Len()
	trace.StderrTruncated = session.stderr.Truncated()
	return trace, nil
}

type stdioSession struct {
	cmd       *exec.Cmd
	cancel    context.CancelFunc
	stdin     io.WriteCloser
	responses chan rpcResponse
	waitCh    chan error
	stderr    *boundedBuffer
	stopOnce  sync.Once
	nextID    int
}

func startSTDIOSession(args []string) (*stdioSession, error) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, err
	}

	session := &stdioSession{
		cmd:       cmd,
		cancel:    cancel,
		stdin:     stdin,
		responses: make(chan rpcResponse, 16),
		waitCh:    make(chan error, 1),
		stderr:    newBoundedBuffer(maxStderrBytes),
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}

	go session.scanStdout(stdout)
	go copyBoundedAndDrain(session.stderr, stderr)
	go func() {
		session.waitCh <- cmd.Wait()
	}()

	return session, nil
}

func (s *stdioSession) runProbe(probe stdioProbe, timeout time.Duration) STDIOObservation {
	s.nextID++
	obs := STDIOObservation{
		Probe:        probe.name,
		RPCMethod:    probe.method,
		ReadOnly:     probe.readOnly,
		Mutating:     probe.mutating,
		MetaIncluded: hasMeta(probe.params),
	}

	request := rpcRequest{JSONRPC: "2.0", ID: s.nextID, Method: probe.method, Params: probe.params}
	payload, err := json.Marshal(request)
	if err != nil {
		obs.ProcessError = true
		return obs
	}
	payload = append(payload, '\n')
	if _, err := s.stdin.Write(payload); err != nil {
		obs.ProcessError = true
		return obs
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case resp, ok := <-s.responses:
			if !ok {
				obs.ProcessError = true
				return obs
			}
			if !sameJSONRPCID(resp.ID, request.ID) {
				continue
			}
			if resp.Error != nil {
				obs.HasRPCError = true
				obs.RPCErrorCode = resp.Error.Code
				return obs
			}
			if len(resp.Result) > 0 {
				obs.HasResult = true
				var result map[string]any
				if err := json.Unmarshal(resp.Result, &result); err == nil {
					obs.Result = result
				}
			}
			return obs
		case <-timer.C:
			obs.Timeout = true
			s.stop()
			return obs
		}
	}
}

func (s *stdioSession) scanStdout(stdout io.Reader) {
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var resp rpcResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
		}
		s.responses <- resp
	}
}

func (s *stdioSession) stop() {
	s.stopOnce.Do(func() {
		_ = s.stdin.Close()
		s.cancel()
		select {
		case <-s.waitCh:
		case <-time.After(500 * time.Millisecond):
			if s.cmd.Process != nil {
				_ = s.cmd.Process.Kill()
			}
			<-s.waitCh
		}
	})
}

func defaultSTDIOProbes(specTarget string) []stdioProbe {
	return []stdioProbe{
		{name: "tools-list", method: "tools/list", params: paramsWithMeta(specTarget), readOnly: true},
		{name: "tools-list-missing-meta", method: "tools/list", params: map[string]any{}, readOnly: true},
		{name: "resources-list", method: "resources/list", params: paramsWithMeta(specTarget), readOnly: true},
		{name: "prompts-list", method: "prompts/list", params: paramsWithMeta(specTarget), readOnly: true},
	}
}

func legacyInitializeParams(specTarget string) map[string]any {
	return map[string]any{
		"protocolVersion": specTarget,
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "mcp-migrate", "version": "0.1.0"},
	}
}

func sameJSONRPCID(got any, want int) bool {
	switch v := got.(type) {
	case float64:
		return int(v) == want
	case int:
		return v == want
	default:
		return fmt.Sprint(v) == fmt.Sprint(want)
	}
}

type boundedBuffer struct {
	mu        sync.Mutex
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func newBoundedBuffer(limit int) *boundedBuffer {
	return &boundedBuffer{limit: limit}
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	remaining := b.limit - b.buf.Len()
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if len(p) > remaining {
		b.buf.Write(p[:remaining])
		b.truncated = true
		return len(p), nil
	}
	b.buf.Write(p)
	return len(p), nil
}

func (b *boundedBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Len()
}

func (b *boundedBuffer) Truncated() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.truncated
}

func copyBoundedAndDrain(dst io.Writer, src io.Reader) {
	_, _ = io.Copy(dst, src)
}

func splitCommandLine(command string) ([]string, error) {
	var args []string
	var current strings.Builder
	var quote rune
	escaped := false
	for _, r := range command {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
		case r == ' ' || r == '\t' || r == '\n':
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if escaped {
		return nil, fmt.Errorf("unterminated escape in server command")
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote in server command")
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args, nil
}

func sanitizeCommand(args []string) string {
	out := append([]string(nil), args...)
	for i := range out {
		if isSensitiveName(out[i]) {
			out[i] = redactArg(out[i])
			if !strings.Contains(out[i], "=") && i+1 < len(out) {
				out[i+1] = "redacted"
			}
			continue
		}
		if strings.Contains(out[i], "=") {
			parts := strings.SplitN(out[i], "=", 2)
			if isSensitiveName(parts[0]) {
				out[i] = parts[0] + "=redacted"
			}
		}
	}
	return strings.Join(out, " ")
}

func redactArg(arg string) string {
	if strings.Contains(arg, "=") {
		parts := strings.SplitN(arg, "=", 2)
		return parts[0] + "=redacted"
	}
	return arg
}
