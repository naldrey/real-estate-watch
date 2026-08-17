package config

import (
	"os"
	"testing"
)

func TestLoadDotEnv(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	content := "FOO=from-file\n" +
		"# a comment\n" +
		"\n" +
		"export BAR=\"quoted value\"\n" +
		"BAZ=unset-in-real-env\n"
	if err := os.WriteFile(".env", []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write .env: %v", err)
	}

	os.Unsetenv("FOO")
	os.Unsetenv("BAR")
	os.Unsetenv("BAZ")
	t.Cleanup(func() {
		os.Unsetenv("BAR")
		os.Unsetenv("BAZ")
	})
	t.Setenv("FOO", "from-real-env") // pre-set: .env must not override this

	if err := loadDotEnv(); err != nil {
		t.Fatalf("loadDotEnv() returned error: %v", err)
	}

	if got, want := os.Getenv("FOO"), "from-real-env"; got != want {
		t.Errorf("FOO = %q, want %q (real env must win over .env)", got, want)
	}
	if got, want := os.Getenv("BAR"), "quoted value"; got != want {
		t.Errorf("BAR = %q, want %q", got, want)
	}
	if got, want := os.Getenv("BAZ"), "unset-in-real-env"; got != want {
		t.Errorf("BAZ = %q, want %q", got, want)
	}
}

func TestLoadDotEnv_MissingFileIsNotAnError(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := loadDotEnv(); err != nil {
		t.Fatalf("loadDotEnv() returned error for a missing .env: %v", err)
	}
}
