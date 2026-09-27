//go:build !darwin && !linux

package jobs

import (
	"os"
	"runtime"
	"testing"
)

// openPTY is the compile-safe stand-in for platforms with no pseudo-terminal
// allocation implemented here. Its only job is to let the cmd/jobs test binary
// BUILD everywhere .goreleaser.yml declares a release target — windows most of
// all — so a contributor on such a platform can run `go test ./...` at all.
//
// It is deliberately fatal and deliberately NOT a skip. A skip would quietly
// retire TestBulkDeleteDeclined, the only test in this package that can catch a
// confirmation prompt replaced by an unconditional true, on precisely the
// platforms this split was added to support — trading a loud compile error for
// a silent coverage hole. Whoever hits this should add that platform's ioctl
// requests alongside darwin and linux in delete_pty_unix_test.go.
func openPTY(t *testing.T) (master, slave *os.File) {
	t.Helper()
	t.Fatalf("no pseudo-terminal allocation implemented for GOOS %q — "+
		"add its ioctl requests rather than skipping the declined-prompt test", runtime.GOOS)
	return nil, nil
}
