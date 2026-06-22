package main

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binPath string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "tt-test-*")
	if err != nil {
		log.Fatalf("failed to create temp dir: %v", err)
	}

	binPath = filepath.Join(tmpDir, "tt")

	cmd := exec.Command("go", "build", "-o", binPath, ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Fatalf("failed to compile tt binary: %v\nOutput: %s", err, string(output))
	}

	code := m.Run()

	os.RemoveAll(tmpDir)
	os.Exit(code)
}

func TestIntegration_BinaryExists(t *testing.T) {
	if binPath == "" {
		t.Fatal("binPath is not set")
	}
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("compiled binary does not exist at %s: %v", binPath, err)
	}
}

func runTT(t *testing.T, home, dbPath, stdin string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Env = append(os.Environ(), "HOME="+home, "TT_DB_PATH="+dbPath)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Run()
	return stdoutBuf.String(), stderrBuf.String(), err
}

func TestIntegration_RunTTHelper(t *testing.T) {
	home := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	stdout, stderr, err := runTT(t, home, dbPath, "", "version")
	if err != nil {
		t.Fatalf("runTT failed: %v, stderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "dev") {
		t.Errorf("expected version output 'dev', got: %s", stdout)
	}
}
