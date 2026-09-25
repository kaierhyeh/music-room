//go:build e2e

package e2e

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func gitLsFiles(t *testing.T, args ...string) []string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", "..", "ls-files"}, args...)...).Output()
	if err != nil {
		t.Skipf("not a git checkout: %v", err)
	}
	return strings.Fields(string(out))
}

func TestRepo_EnvFilesAreIgnoredAndNeverCommitted(t *testing.T) {
	for _, f := range gitLsFiles(t, "--full-name", "--", "..") {
		base := f[strings.LastIndex(f, "/")+1:]
		if strings.HasPrefix(base, ".env") && !strings.HasSuffix(base, ".example") {
			t.Errorf("%s is tracked by git", f)
		}
	}
	for _, f := range []string{".env", ".env.smtp", "../musicroom-app/.env"} {
		if err := exec.Command("git", "-C", "..", "check-ignore", "-q", f).Run(); err != nil {
			t.Errorf("%s is not git-ignored", f)
		}
	}
}

func TestRepo_DependenciesAreDownloadedNotCommitted(t *testing.T) {
	for _, f := range gitLsFiles(t, "--full-name", "--", "..") {
		if strings.Contains(f, "node_modules/") || strings.HasPrefix(f, "musicroom-backend/vendor/") {
			t.Errorf("third-party code is committed: %s", f)
		}
	}
	for _, f := range []string{"../go.mod", "../go.sum", "../../musicroom-app/package.json", "../../musicroom-app/package-lock.json"} {
		if _, err := os.Stat(f); err != nil {
			t.Errorf("dependency manifest %s is missing", f)
		}
	}
	mk, err := os.ReadFile("../Makefile")
	if err != nil || !strings.Contains(string(mk), "go mod download") {
		t.Error("the Makefile must download the Go dependencies")
	}
}
