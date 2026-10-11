//go:build unix

package output

import (
	"bytes"
	"os"
	"os/signal"
	"testing"
	"time"

	"github.com/charmbracelet/colorprofile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const signalWait = 5 * time.Second

func interruptSelf(t *testing.T) {
	t.Helper()
	self, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	require.NoError(t, self.Signal(os.Interrupt))
}

func drain(ch chan os.Signal) {
	for len(ch) > 0 {
		<-ch
	}
}

func waitForPagerExit(t *testing.T, p *pagedWriter) {
	t.Helper()
	select {
	case <-p.exited:
	case <-time.After(signalWait):
		t.Fatal("pager did not exit")
	}
}

func assertInterruptsReleased(t *testing.T, p *pagedWriter) {
	t.Helper()
	drain(p.interrupts)
	observer := make(chan os.Signal, 1)
	signal.Notify(observer, os.Interrupt)
	defer signal.Stop(observer)

	interruptSelf(t)

	select {
	case <-observer:
	case <-time.After(signalWait):
		t.Fatal("interrupt not delivered")
	}
	assert.Empty(t, p.interrupts, "pager still captures Ctrl-C after it exited")
}

func TestPagedWriter_CapturesInterruptWhilePagerRuns(t *testing.T) {
	var out bytes.Buffer
	p := newPagedWriter(helperPager(t, "copy"), os.Environ(), &out, colorprofile.TrueColor)
	_, err := p.Write([]byte("row\n"))
	require.NoError(t, err)

	interruptSelf(t)

	select {
	case <-p.interrupts:
	case <-time.After(signalWait):
		t.Fatal("interrupt not captured while the pager runs")
	}
	require.NoError(t, p.Close())
	assert.Equal(t, "row\n", out.String())
	assertInterruptsReleased(t, p)
}

func TestPagedWriter_ReleasesInterruptWhenPagerQuitsBeforeClose(t *testing.T) {
	p := newPagedWriter(helperPager(t, "quit"), os.Environ(), &bytes.Buffer{}, colorprofile.TrueColor)
	_, err := p.Write([]byte("row\n"))
	require.NoError(t, err)

	waitForPagerExit(t, p)

	assertInterruptsReleased(t, p)
	assert.NoError(t, p.Close())
}
