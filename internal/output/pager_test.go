package output

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const pagerModeEnv = "REVENIUM_TEST_PAGER_MODE"

func TestPagerHelperProcess(t *testing.T) {
	switch os.Getenv(pagerModeEnv) {
	case "copy":
		_, _ = io.Copy(os.Stdout, os.Stdin)
		os.Exit(0)
	case "quit":
		os.Exit(0)
	}
}

func helperPager(t *testing.T, mode string) []string {
	t.Helper()
	t.Setenv(pagerModeEnv, mode)
	return []string{os.Args[0], "-test.run=^TestPagerHelperProcess$"}
}

func lookupFrom(env map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		value, ok := env[name]
		return value, ok
	}
}

func TestPagerCommand(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want []string
	}{
		{"defaults to less", map[string]string{}, []string{"less", "-FRX"}},
		{"REVENIUM_PAGER with args", map[string]string{"REVENIUM_PAGER": "most -s"}, []string{"most", "-s"}},
		{"REVENIUM_PAGER cat disables", map[string]string{"REVENIUM_PAGER": "cat"}, nil},
		{"REVENIUM_PAGER empty disables", map[string]string{"REVENIUM_PAGER": ""}, nil},
		{"REVENIUM_PAGER blank disables", map[string]string{"REVENIUM_PAGER": "  "}, nil},
		{"PAGER used when REVENIUM_PAGER unset", map[string]string{"PAGER": "more"}, []string{"more"}},
		{"PAGER cat disables", map[string]string{"PAGER": "cat"}, nil},
		{"PAGER empty disables", map[string]string{"PAGER": ""}, nil},
		{"REVENIUM_PAGER beats PAGER cat", map[string]string{"REVENIUM_PAGER": "less -R", "PAGER": "cat"}, []string{"less", "-R"}},
		{"REVENIUM_PAGER cat beats PAGER", map[string]string{"REVENIUM_PAGER": "cat", "PAGER": "less"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, pagerCommand(lookupFrom(tt.env)))
		})
	}
}

func TestPagerEnv(t *testing.T) {
	assert.Equal(t, []string{"HOME=/h", "LESS=FRX"}, pagerEnv([]string{"HOME=/h"}))
	assert.Equal(t, []string{"LESS=-R"}, pagerEnv([]string{"LESS=-R"}))
	assert.Equal(t, []string{"LESSOPEN=x", "LESS=FRX"}, pagerEnv([]string{"LESSOPEN=x"}))
}

func renderSample(f *Formatter) error {
	if err := f.RenderJSON(map[string]string{"id": "src-1", "status": "active"}); err != nil {
		return err
	}
	def := TableDef{Headers: []string{"ID", "Status"}, StatusColumn: 1}
	return f.RenderTable(def, [][]string{{"src-1", "active"}, {"src-2", "inactive"}})
}

func renderThroughPager(t *testing.T, profile colorprofile.Profile) string {
	t.Helper()
	var out bytes.Buffer
	f := NewWithWriter(io.Discard, io.Discard, false, false)
	f.usePager(newPagedWriter(helperPager(t, "copy"), os.Environ(), &out, profile))
	require.NoError(t, renderSample(f))
	require.NoError(t, f.Close())
	return out.String()
}

func TestPagedWriter_RoundTripMatchesDirectOutput(t *testing.T) {
	var direct bytes.Buffer
	require.NoError(t, renderSample(NewWithWriter(&direct, io.Discard, false, false)))

	paged := renderThroughPager(t, colorprofile.TrueColor)

	assert.Equal(t, direct.String(), paged)
	assert.Contains(t, paged, "\x1b[")
}

func TestPagedWriter_NoTTYProfileStripsANSI(t *testing.T) {
	paged := renderThroughPager(t, colorprofile.NoTTY)

	assert.Contains(t, paged, "src-2")
	assert.NotContains(t, paged, "\x1b")
}

func TestPagedWriter_MissingBinaryFallsBackToOut(t *testing.T) {
	var out bytes.Buffer
	missing := filepath.Join(t.TempDir(), "no-such-pager")
	p := newPagedWriter([]string{missing}, os.Environ(), &out, colorprofile.TrueColor)

	n, err := p.Write([]byte("hello\n"))

	require.NoError(t, err)
	assert.Equal(t, 6, n)
	assert.Equal(t, "hello\n", out.String())
	assert.NoError(t, p.Close())
}

func TestPagedWriter_EarlyQuitSwallowsBrokenPipe(t *testing.T) {
	var out bytes.Buffer
	p := newPagedWriter(helperPager(t, "quit"), os.Environ(), &out, colorprofile.TrueColor)
	large := bytes.Repeat([]byte("x"), 1<<20)

	n, err := p.Write(large)
	require.NoError(t, err)
	assert.Equal(t, len(large), n)

	n, err = p.Write([]byte("more\n"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)

	assert.NoError(t, p.Close())
}

func TestPagedWriter_LazyStart(t *testing.T) {
	p := newPagedWriter(helperPager(t, "copy"), os.Environ(), &bytes.Buffer{}, colorprofile.TrueColor)

	assert.Nil(t, p.cmd)
	assert.NoError(t, p.Close())
	assert.Nil(t, p.cmd)
}

func TestPagedWriter_CloseIsIdempotent(t *testing.T) {
	var out bytes.Buffer
	p := newPagedWriter(helperPager(t, "copy"), os.Environ(), &out, colorprofile.TrueColor)
	_, err := p.Write([]byte("once\n"))
	require.NoError(t, err)

	require.NoError(t, p.Close())
	assert.NoError(t, p.Close())
	assert.Equal(t, "once\n", out.String())
}

func TestFollower_WritesDirectlyBeforePagerStarts(t *testing.T) {
	var out bytes.Buffer
	p := newPagedWriter(helperPager(t, "copy"), os.Environ(), &out, colorprofile.TrueColor)

	_, err := p.follower().Write([]byte("Deleted 1 job.\n"))

	require.NoError(t, err)
	assert.Equal(t, "Deleted 1 job.\n", out.String())
	assert.Nil(t, p.cmd)
	assert.NoError(t, p.Close())
}

func TestFollower_JoinsPagerStreamAfterFirstWrite(t *testing.T) {
	var out bytes.Buffer
	p := newPagedWriter(helperPager(t, "copy"), os.Environ(), &out, colorprofile.TrueColor)
	follower := p.follower()

	_, err := follower.Write([]byte("a\n"))
	require.NoError(t, err)
	_, err = p.Write([]byte("b\n"))
	require.NoError(t, err)
	_, err = follower.Write([]byte("c\n"))
	require.NoError(t, err)

	require.NoError(t, p.Close())
	assert.Equal(t, "a\nb\nc\n", out.String())
}

func TestEnablePager_NoOpForNewWithWriter(t *testing.T) {
	var buf bytes.Buffer
	f := NewWithWriter(&buf, io.Discard, false, false)

	follower, paging := f.EnablePager()

	assert.Nil(t, follower)
	assert.False(t, paging)
	assert.Same(t, &buf, f.Writer())
}

func TestEnablePager_NoOpWhenQuiet(t *testing.T) {
	t.Setenv("REVENIUM_PAGER", strings.Join(helperPager(t, "copy"), " "))
	f := &Formatter{isTTY: true, quiet: true}

	follower, paging := f.EnablePager()

	assert.Nil(t, follower)
	assert.False(t, paging)
}

func TestEnablePager_NoOpWhenPagerIsCat(t *testing.T) {
	t.Setenv("REVENIUM_PAGER", "cat")
	f := &Formatter{isTTY: true}

	follower, paging := f.EnablePager()

	assert.Nil(t, follower)
	assert.False(t, paging)
}

func TestEnablePager_InstallsLazyPagerOnTTY(t *testing.T) {
	t.Setenv("REVENIUM_PAGER", strings.Join(helperPager(t, "copy"), " "))
	f := &Formatter{isTTY: true}

	follower, paging := f.EnablePager()

	assert.NotNil(t, follower)
	assert.True(t, paging)
	require.NotNil(t, f.pager)
	assert.Nil(t, f.pager.cmd)
	assert.NoError(t, f.Close())
}
