package output

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	"github.com/charmbracelet/colorprofile"
	"github.com/mattn/go-shellwords"
)

var pagerEnvVars = []string{"REVENIUM_PAGER", "PAGER"}

var defaultPagerCommand = []string{"less", "-FRX"}

const disabledPagerValue = "cat"

const lessEnvPrefix = "LESS="

const defaultLessOptions = lessEnvPrefix + "FRX"

func pagerCommand(lookupEnv func(string) (string, bool)) []string {
	for _, name := range pagerEnvVars {
		if value, ok := lookupEnv(name); ok {
			return parsePagerValue(value)
		}
	}
	return append([]string(nil), defaultPagerCommand...)
}

func parsePagerValue(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" || value == disabledPagerValue {
		return nil
	}
	argv, err := shellwords.Parse(value)
	if err != nil || len(argv) == 0 {
		return nil
	}
	return argv
}

func pagerEnv(environ []string) []string {
	for _, entry := range environ {
		if strings.HasPrefix(entry, lessEnvPrefix) {
			return environ
		}
	}
	return append(environ[:len(environ):len(environ)], defaultLessOptions)
}

type pagedWriter struct {
	argv    []string
	env     []string
	out     io.Writer
	profile colorprofile.Profile

	target     io.Writer
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	interrupts chan os.Signal
	exited     chan struct{}
	waitErr    error
	closed     bool
}

func newPagedWriter(argv, env []string, out io.Writer, profile colorprofile.Profile) *pagedWriter {
	return &pagedWriter{argv: argv, env: env, out: out, profile: profile}
}

func (p *pagedWriter) Write(b []byte) (int, error) {
	if p.target == nil {
		p.target = p.start()
	}
	return p.target.Write(b)
}

func (p *pagedWriter) start() io.Writer {
	cmd := exec.Command(p.argv[0], p.argv[1:]...)
	cmd.Stdout = p.out
	cmd.Stderr = os.Stderr
	cmd.Env = p.env

	stdin, err := cmd.StdinPipe()
	if err == nil {
		err = cmd.Start()
	}
	if err != nil {
		return p.colorWriter(p.out)
	}

	p.cmd = cmd
	p.stdin = stdin
	p.captureInterrupts()
	p.exited = make(chan struct{})
	go p.releaseInterruptsOnExit()
	return p.colorWriter(&pagerInput{pipe: stdin})
}

// Ctrl-C reaches the whole foreground process group and less handles it itself;
// if the CLI died, the shell would take the terminal back while less still draws.
func (p *pagedWriter) captureInterrupts() {
	p.interrupts = make(chan os.Signal, 1)
	signal.Notify(p.interrupts, os.Interrupt)
}

// Quitting the pager early must hand Ctrl-C back even while the command keeps
// working, rather than only once Close runs after the command returns.
func (p *pagedWriter) releaseInterruptsOnExit() {
	p.waitErr = p.cmd.Wait()
	signal.Stop(p.interrupts)
	close(p.exited)
}

func (p *pagedWriter) colorWriter(forward io.Writer) io.Writer {
	return &colorprofile.Writer{Forward: forward, Profile: p.profile}
}

func (p *pagedWriter) follower() io.Writer {
	return pagerFollower{pager: p}
}

func (p *pagedWriter) Close() error {
	if p.cmd == nil || p.closed {
		return nil
	}
	p.closed = true

	_ = p.stdin.Close()
	<-p.exited

	if p.waitErr != nil {
		return fmt.Errorf("pager %s: %w", p.argv[0], p.waitErr)
	}
	return nil
}

type pagerInput struct {
	pipe io.Writer
	gone bool
}

// A write into the pager's pipe only fails once the pager has exited (the user quit
// early: EPIPE on Unix, ERROR_NO_DATA/ERROR_BROKEN_PIPE on Windows), so the rest has no reader.
func (w *pagerInput) Write(b []byte) (int, error) {
	if !w.gone {
		if _, err := w.pipe.Write(b); err != nil {
			w.gone = true
		}
	}
	return len(b), nil
}

type pagerFollower struct {
	pager *pagedWriter
}

func (f pagerFollower) Write(b []byte) (int, error) {
	if f.pager.target == nil {
		return f.pager.out.Write(b)
	}
	return f.pager.Write(b)
}

// EnablePager routes Formatter output through the user's pager when stdout is a
// terminal. It returns a writer that keeps other stdout text in order with the paged output.
func (f *Formatter) EnablePager() (io.Writer, bool) {
	if !f.isTTY || f.quiet {
		return nil, false
	}
	argv := pagerCommand(os.LookupEnv)
	if argv == nil {
		return nil, false
	}

	// The pager's pipe is not a terminal, so a profile detected from it would strip all ANSI.
	profile := colorprofile.Detect(os.Stdout, os.Environ())
	p := newPagedWriter(argv, pagerEnv(os.Environ()), os.Stdout, profile)
	f.usePager(p)
	return p.follower(), true
}

func (f *Formatter) usePager(p *pagedWriter) {
	f.writer = p
	f.pager = p
}

// Close waits for the pager, if one was started, to exit.
func (f *Formatter) Close() error {
	if f.pager == nil {
		return nil
	}
	return f.pager.Close()
}
