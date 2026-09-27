package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// requireShellToolsEnv names the environment variable that turns "tool not
// installed" from a skip into a failure. See requireTool, and
// TestRequireShellToolsEnvNameIsWired for why the name itself is load-bearing.
const requireShellToolsEnv = "KROOT_REQUIRE_SHELL_TOOLS"

// requireTool skips when one of the external tools these tests shell out to is
// absent, and fails when the environment says the tool is required.
//
// Both halves matter, for opposite reasons. On a developer machine a skip is
// correct: mandoc is installed nowhere by default, and demanding it would make
// `go test ./...` fail for a reason that has nothing to do with the code. In CI a
// skip is wrong, because the tests that use the tool silently do not run — and a
// skip is indistinguishable from a pass in a required status check, so the roff
// and shell validation would appear to be enforced while covering nothing. That
// is the same class of drift this repository keeps having to correct in its
// documents, so the tool's presence is asserted where it is actually relied upon.
func requireTool(t *testing.T, name string) {
	t.Helper()

	if _, err := exec.LookPath(name); err == nil {
		return
	} else if os.Getenv(requireShellToolsEnv) != "" {
		t.Fatalf("%s is required by %s but is not installed: %v", name, t.Name(), err)
	} else {
		t.Skipf("%s is not installed: %v", name, err)
	}
}

// TestRequireShellToolsEnvNameIsWired guards the seam between this constant and
// the CI workflow that sets it.
//
// The two are a literal in a Go file and a literal in a YAML file, and nothing
// else connects them. A rename on either side leaves the tests skipping in CI
// again — the exact condition this mechanism exists to remove, restored by the
// change meant to prevent it, and visible nowhere: the suite passes, mandoc is
// installed, and the roff validation quietly covers nothing again.
//
// So the workflow is read and the names compared. It is worth a test that reads a
// file to catch a string that no compiler can check. This mismatch did occur while
// writing the change, which is how the test came to exist.
func TestRequireShellToolsEnvNameIsWired(t *testing.T) {
	workflow, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ci.yaml"))
	if err != nil {
		t.Fatalf("read ci.yaml: %v", err)
	}

	if !strings.Contains(string(workflow), requireShellToolsEnv+":") {
		t.Errorf("ci.yaml does not set %s; requireTool would skip in CI instead of failing, and the roff and shell validation would stop being enforced", requireShellToolsEnv)
	}
}

// TestRequireToolSkipsWhenTheToolIsAbsent pins the developer-facing half of the
// contract. mandoc is installed nowhere by default, so requiring it
// unconditionally would make `go test ./...` fail for a reason unrelated to the
// code. The failing half is exercised by running the compiled suite with the
// variable set and the tool removed from PATH, which is what CI does.
func TestRequireToolSkipsWhenTheToolIsAbsent(t *testing.T) {
	t.Run("absent tool skips", func(t *testing.T) {
		// A name that cannot resolve, standing in for mandoc on a machine without it.
		requireTool(t, "kroot-no-such-tool-for-testing")
		t.Error("requireTool returned instead of skipping; the suite would fail on a machine without mandoc")
	})
}
