//go:build integration

package tests

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestFakeFirebaseIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	apiCmd := exec.CommandContext(ctx, "go", "run", "./cmd/api")
	apiCmd.Dir = ".."
	apiCmd.Stdout = os.Stdout
	apiCmd.Stderr = os.Stderr
	if err := apiCmd.Start(); err != nil {
		t.Fatalf("start api failed: %v", err)
	}

	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/fakefirebasesub")
	cmd.Dir = ".."
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// fakefirebasesub should exit successfully; fail if it doesn't
		_ = apiCmd.Process.Kill()
		_ = apiCmd.Wait()
		t.Fatalf("fakefirebasesub failed: %v", err)
	}

	_ = apiCmd.Process.Kill()
	_ = apiCmd.Wait()
}
