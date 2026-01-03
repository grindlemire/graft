package graft

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestReproGoList(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create go.mod
	cwd, _ := os.Getwd()
	modContent := fmt.Sprintf(`module testmodule
go 1.25
require github.com/grindlemire/graft v0.0.0
replace github.com/grindlemire/graft => %s
`, cwd)
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(modContent), 0644)

	// Create test.go
	code := `package testmodule

import (
	"context"
	"github.com/grindlemire/graft"
)

var node = graft.Node[string]{
	ID: "testnode",
	Run: func(ctx context.Context) (string, error) { return "ok", nil },
}
`
	os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte(code), 0644)

	// Run go list with -mod=mod
	cmd := exec.Command("go", "list", "-mod=mod", "-json", ".")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("go list output:\n%s", string(out))
	if err != nil {
		t.Errorf("go list failed: %v", err)
	}

	// Run go list ./... with -mod=mod
	cmd2 := exec.Command("go", "list", "-mod=mod", "-json", "./...")
	cmd2.Dir = tmpDir
	out2, err2 := cmd2.CombinedOutput()
	t.Logf("go list ./... output:\n%s", string(out2))
	if err2 != nil {
		t.Errorf("go list ./... failed: %v", err2)
	}
}
