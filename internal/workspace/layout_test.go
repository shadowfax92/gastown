package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/steveyegge/gastown/internal/config"
)

func TestResolveProjectLayoutDisabledByDefault(t *testing.T) {
	townRoot := t.TempDir()
	rigPath := filepath.Join(townRoot, "gastown")
	if err := os.MkdirAll(rigPath, 0755); err != nil {
		t.Fatalf("mkdir rig: %v", err)
	}

	layout, err := ResolveProjectLayout(townRoot, "gastown", rigPath)
	if err != nil {
		t.Fatalf("ResolveProjectLayout: %v", err)
	}
	if layout.Enabled {
		t.Fatalf("layout should be disabled without workspace.root: %+v", layout)
	}
}

func TestEnsureProjectLayoutCreatesIdempotentSymlink(t *testing.T) {
	townRoot := t.TempDir()
	rigPath := filepath.Join(townRoot, "gastown")
	llmRoot := filepath.Join(townRoot, "llm")
	if err := os.MkdirAll(rigPath, 0755); err != nil {
		t.Fatalf("mkdir rig: %v", err)
	}

	settings := config.NewTownSettings()
	settings.Workspace = &config.WorkspaceConfig{Root: llmRoot}
	if err := config.SaveTownSettings(config.TownSettingsPath(townRoot), settings); err != nil {
		t.Fatalf("SaveTownSettings: %v", err)
	}

	layout, err := EnsureProjectLayout(townRoot, "Gas Town", rigPath)
	if err != nil {
		t.Fatalf("EnsureProjectLayout: %v", err)
	}
	if !layout.Enabled {
		t.Fatal("layout should be enabled")
	}
	wantProjectRoot := filepath.Join(llmRoot, "gas-town")
	if layout.ProjectRoot != wantProjectRoot {
		t.Fatalf("ProjectRoot = %q, want %q", layout.ProjectRoot, wantProjectRoot)
	}
	if _, err := os.Stat(wantProjectRoot); err != nil {
		t.Fatalf("project root not created: %v", err)
	}

	linkPath := filepath.Join(rigPath, DefaultProjectSymlink)
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink: %v", err)
	}
	if target != wantProjectRoot {
		t.Fatalf("symlink target = %q, want %q", target, wantProjectRoot)
	}

	if _, err := EnsureProjectLayout(townRoot, "Gas Town", rigPath); err != nil {
		t.Fatalf("EnsureProjectLayout second call: %v", err)
	}
}

func TestEnsureProjectLayoutRejectsUnsafeExistingPath(t *testing.T) {
	townRoot := t.TempDir()
	rigPath := filepath.Join(townRoot, "gastown")
	llmRoot := filepath.Join(townRoot, "llm")
	if err := os.MkdirAll(filepath.Join(rigPath, DefaultProjectSymlink), 0755); err != nil {
		t.Fatalf("mkdir conflicting .llm: %v", err)
	}

	settings := config.NewTownSettings()
	settings.Workspace = &config.WorkspaceConfig{Root: llmRoot}
	if err := config.SaveTownSettings(config.TownSettingsPath(townRoot), settings); err != nil {
		t.Fatalf("SaveTownSettings: %v", err)
	}

	_, err := EnsureProjectLayout(townRoot, "gastown", rigPath)
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !strings.Contains(err.Error(), "not a symlink") {
		t.Fatalf("error = %v, want not-a-symlink conflict", err)
	}
}

func TestResolveProjectLayoutExpandsRelativeRootAndCustomSymlink(t *testing.T) {
	townRoot := t.TempDir()
	rigPath := filepath.Join(townRoot, "gastown")
	if err := os.MkdirAll(rigPath, 0755); err != nil {
		t.Fatalf("mkdir rig: %v", err)
	}

	settings := config.NewTownSettings()
	settings.Workspace = &config.WorkspaceConfig{
		Root:           "workspaces",
		ProjectSymlink: ".review",
	}
	if err := config.SaveTownSettings(config.TownSettingsPath(townRoot), settings); err != nil {
		t.Fatalf("SaveTownSettings: %v", err)
	}

	layout, err := ResolveProjectLayout(townRoot, "gastown", rigPath)
	if err != nil {
		t.Fatalf("ResolveProjectLayout: %v", err)
	}
	if layout.Root != filepath.Join(townRoot, "workspaces") {
		t.Fatalf("Root = %q, want town-relative path", layout.Root)
	}
	if layout.LinkPath != filepath.Join(rigPath, ".review") {
		t.Fatalf("LinkPath = %q, want custom symlink", layout.LinkPath)
	}
}
