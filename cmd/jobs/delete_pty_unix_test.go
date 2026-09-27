//go:build darwin || linux

package jobs

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

// openPTY allocates a pseudo-terminal and returns its master and slave ends.
//
// WHY A REAL TERMINAL IS NEEDED AT ALL: resource.ConfirmDelete decides whether
// to prompt with term.IsTerminal(os.Stdin.Fd()) and then reads the answer from
// os.Stdin itself. Neither is injectable, and the shared helper may not be
// forked, wrapped or replaced. So the only way to reach its INTERACTIVE rung
// from a test — the one rung where an operator can say no — is to hand it a
// terminal. A pipe is never one (see withNonTTYStdin), which is exactly why the
// non-terminal confirm case cannot catch a ConfirmDelete replaced by an
// unconditional true, and why TestBulkDeleteDeclined exists separately.
//
// The ioctl requests are written as literals rather than syscall constants
// because the constant names differ between darwin and linux, and this one file
// serves both. The build constraint above exists for a separate reason: the
// ioctl MECHANISM itself has no counterpart outside these two platforms, so a
// GOOS without one compiles the deliberately fatal stub in
// delete_pty_other_test.go instead of failing the whole package's build (see
// 28-REVIEW.md CR-01 — .goreleaser.yml declares windows a release target). The
// values below are the standard ones for darwin (developer machines) and
// linux/amd64 (CI — .github/workflows/go-test.yml runs on ubuntu-latest).
func openPTY(t *testing.T) (master, slave *os.File) {
	t.Helper()

	m, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	require.NoError(t, err, "opening /dev/ptmx to allocate a pseudo-terminal")

	var name string
	switch runtime.GOOS {
	case "darwin":
		const (
			tiocPtyGrant = 0x20007454 // TIOCPTYGRANT
			tiocPtyUnlk  = 0x20007452 // TIOCPTYUNLK
			tiocPtyGname = 0x40807453 // TIOCPTYGNAME, fills a 128-byte name
		)
		if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, m.Fd(), tiocPtyGrant, 0); e != 0 {
			require.FailNowf(t, "TIOCPTYGRANT failed", "%v", e)
		}
		if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, m.Fd(), tiocPtyUnlk, 0); e != 0 {
			require.FailNowf(t, "TIOCPTYUNLK failed", "%v", e)
		}
		buf := make([]byte, 128)
		if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, m.Fd(), tiocPtyGname,
			uintptr(unsafe.Pointer(&buf[0]))); e != 0 {
			require.FailNowf(t, "TIOCPTYGNAME failed", "%v", e)
		}
		if i := bytes.IndexByte(buf, 0); i >= 0 {
			buf = buf[:i]
		}
		name = string(buf)
	case "linux":
		const (
			tiocSPtLck = 0x40045431 // TIOCSPTLCK, unlock the slave
			tiocGPtN   = 0x80045430 // TIOCGPTN, read the slave's number
		)
		var unlock int32
		if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, m.Fd(), tiocSPtLck,
			uintptr(unsafe.Pointer(&unlock))); e != 0 {
			require.FailNowf(t, "TIOCSPTLCK failed", "%v", e)
		}
		var n uint32
		if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, m.Fd(), tiocGPtN,
			uintptr(unsafe.Pointer(&n))); e != 0 {
			require.FailNowf(t, "TIOCGPTN failed", "%v", e)
		}
		name = fmt.Sprintf("/dev/pts/%d", n)
	default:
		// Unreachable under this file's build constraint, and deliberately
		// fatal rather than skipped anyway. A skip here would quietly retire
		// the only test that can catch a deleted confirmation prompt.
		t.Fatalf("no pseudo-terminal allocation implemented for GOOS %q — "+
			"add its ioctl requests here rather than skipping the declined-prompt test",
			runtime.GOOS)
	}

	s, err := os.OpenFile(name, os.O_RDWR|syscall.O_NOCTTY, 0)
	require.NoError(t, err, "opening the pseudo-terminal slave %s", name)

	t.Cleanup(func() {
		_ = s.Close()
		_ = m.Close()
	})
	return m, s
}
