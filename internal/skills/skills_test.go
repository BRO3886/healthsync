package skills_test

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/BRO3886/healthsync/internal/skills"
)

func makeTestFS() fstest.MapFS {
	return fstest.MapFS{
		"skills/healthsync/SKILL.md": {
			Data: []byte("# healthsync skill\n"),
		},
		"skills/healthsync/references/schema.md": {
			Data: []byte("# Schema\n"),
		},
	}
}

func TestDefaultTargets(t *testing.T) {
	targets := skills.DefaultTargets("/home/user")
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets))
	}
	if targets[0].Key != "claude" {
		t.Errorf("expected first target key 'claude', got %q", targets[0].Key)
	}
	if targets[1].Key != "codex" {
		t.Errorf("expected second target key 'codex', got %q", targets[1].Key)
	}
}

func TestSkillDir(t *testing.T) {
	target := skills.AgentTarget{
		Name:    "Claude Code",
		Key:     "claude",
		BaseDir: "/home/user/.claude/skills",
	}
	got := skills.SkillDir(target)
	want := "/home/user/.claude/skills/healthsync"
	if got != want {
		t.Errorf("SkillDir = %q, want %q", got, want)
	}
}

func TestInstallAndUninstall(t *testing.T) {
	tmpDir := t.TempDir()
	target := skills.AgentTarget{
		Name:    "Test Agent",
		Key:     "test",
		BaseDir: filepath.Join(tmpDir, "skills"),
	}

	embeddedFS := makeTestFS()

	// Install
	written, err := skills.Install(embeddedFS, target, "v0.3.0")
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	if len(written) == 0 {
		t.Fatal("expected files to be written")
	}

	// Verify SKILL.md exists
	skillMD := filepath.Join(skills.SkillDir(target), "SKILL.md")
	if _, err := os.Stat(skillMD); err != nil {
		t.Errorf("SKILL.md not found after install: %v", err)
	}

	// Verify version file
	if !skills.IsInstalled(target) {
		t.Error("IsInstalled should return true after install")
	}
	if v := skills.InstalledVersion(target); v != "v0.3.0" {
		t.Errorf("InstalledVersion = %q, want %q", v, "v0.3.0")
	}

	// Verify references subdir
	schemaPath := filepath.Join(skills.SkillDir(target), "references", "schema.md")
	if _, err := os.Stat(schemaPath); err != nil {
		t.Errorf("references/schema.md not found after install: %v", err)
	}

	// Uninstall
	removed, err := skills.Uninstall(target)
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}
	if !removed {
		t.Error("Uninstall should return true when skill was installed")
	}

	// Verify gone
	if skills.IsInstalled(target) {
		t.Error("IsInstalled should return false after uninstall")
	}

	// Uninstall again — should return false, no error
	removed, err = skills.Uninstall(target)
	if err != nil {
		t.Fatalf("Second Uninstall failed: %v", err)
	}
	if removed {
		t.Error("Second Uninstall should return false (nothing to remove)")
	}
}

func TestInstallIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	target := skills.AgentTarget{
		Name:    "Test Agent",
		Key:     "test",
		BaseDir: filepath.Join(tmpDir, "skills"),
	}
	embeddedFS := makeTestFS()

	// Install twice — second should overwrite cleanly
	if _, err := skills.Install(embeddedFS, target, "v0.3.0"); err != nil {
		t.Fatalf("first Install failed: %v", err)
	}
	if _, err := skills.Install(embeddedFS, target, "v0.3.1"); err != nil {
		t.Fatalf("second Install failed: %v", err)
	}

	if v := skills.InstalledVersion(target); v != "v0.3.1" {
		t.Errorf("InstalledVersion = %q, want v0.3.1 after re-install", v)
	}
}

func TestIsInstalledFalseWhenMissing(t *testing.T) {
	tmpDir := t.TempDir()
	target := skills.AgentTarget{
		Name:    "Test Agent",
		Key:     "test",
		BaseDir: filepath.Join(tmpDir, "skills"),
	}
	if skills.IsInstalled(target) {
		t.Error("IsInstalled should return false for non-existent directory")
	}
}

func TestInstalledVersionEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	target := skills.AgentTarget{
		Name:    "Test Agent",
		Key:     "test",
		BaseDir: filepath.Join(tmpDir, "skills"),
	}
	if v := skills.InstalledVersion(target); v != "" {
		t.Errorf("InstalledVersion = %q, want empty string for non-existent", v)
	}
}

func TestDetectAgents(t *testing.T) {
	tmpDir := t.TempDir()

	// Create ~/.claude dir but not ~/.agents
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	targets := []skills.AgentTarget{
		{
			Name:    "Claude Code",
			Key:     "claude",
			BaseDir: filepath.Join(claudeDir, "skills"),
		},
		{
			Name:    "Codex CLI",
			Key:     "codex",
			BaseDir: filepath.Join(tmpDir, ".agents", "skills"),
		},
	}

	detected := skills.DetectAgents(targets)
	if len(detected) != 1 {
		t.Fatalf("expected 1 detected agent, got %d", len(detected))
	}
	if detected[0].Key != "claude" {
		t.Errorf("expected claude detected, got %q", detected[0].Key)
	}
}

func TestDisplayPath(t *testing.T) {
	tests := []struct {
		path    string
		homeDir string
		want    string
	}{
		{"/home/user/.claude/skills", "/home/user", "~/.claude/skills"},
		{"/home/user/.claude/skills", "/other", "/home/user/.claude/skills"},
		{"/home/user", "/home/user", "~"},
	}
	for _, tt := range tests {
		got := skills.DisplayPath(tt.path, tt.homeDir)
		if got != tt.want {
			t.Errorf("DisplayPath(%q, %q) = %q, want %q", tt.path, tt.homeDir, got, tt.want)
		}
	}
}
