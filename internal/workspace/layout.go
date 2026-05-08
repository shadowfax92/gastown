package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/steveyegge/gastown/internal/config"
)

const DefaultProjectSymlink = ".llm"

// ProjectLayout describes the optional external workspace layout for a rig.
type ProjectLayout struct {
	Enabled     bool
	Root        string
	ProjectRoot string
	LinkName    string
	LinkPath    string
}

// ResolveProjectLayout resolves the configured workspace layout for a rig.
// If town settings do not configure workspace.root, the returned layout is
// disabled and callers should use the legacy in-rig directories.
func ResolveProjectLayout(townRoot, rigName, rigPath string) (ProjectLayout, error) {
	settings, err := config.LoadOrCreateTownSettings(config.TownSettingsPath(townRoot))
	if err != nil {
		return ProjectLayout{}, fmt.Errorf("loading town settings: %w", err)
	}
	if settings.Workspace == nil || strings.TrimSpace(settings.Workspace.Root) == "" {
		return ProjectLayout{}, nil
	}

	root, err := expandWorkspacePath(settings.Workspace.Root, townRoot)
	if err != nil {
		return ProjectLayout{}, err
	}

	linkName := strings.TrimSpace(settings.Workspace.ProjectSymlink)
	if linkName == "" {
		linkName = DefaultProjectSymlink
	}
	if err := validateSymlinkName(linkName); err != nil {
		return ProjectLayout{}, err
	}

	projectName := stablePathSegment(rigName, "project")
	projectRoot := filepath.Join(root, projectName)
	return ProjectLayout{
		Enabled:     true,
		Root:        root,
		ProjectRoot: projectRoot,
		LinkName:    linkName,
		LinkPath:    filepath.Join(rigPath, linkName),
	}, nil
}

// EnsureProjectLayout creates the configured project workspace root and rig
// symlink. It is safe to call repeatedly. Existing symlinks must point to the
// configured target; files or directories at the symlink path are not replaced.
func EnsureProjectLayout(townRoot, rigName, rigPath string) (ProjectLayout, error) {
	layout, err := ResolveProjectLayout(townRoot, rigName, rigPath)
	if err != nil || !layout.Enabled {
		return layout, err
	}

	if err := os.MkdirAll(layout.ProjectRoot, 0755); err != nil {
		return layout, fmt.Errorf("creating workspace root %s: %w", layout.ProjectRoot, err)
	}

	info, err := os.Lstat(layout.LinkPath)
	if os.IsNotExist(err) {
		if err := os.Symlink(layout.ProjectRoot, layout.LinkPath); err != nil {
			return layout, fmt.Errorf("creating workspace symlink %s -> %s: %w", layout.LinkPath, layout.ProjectRoot, err)
		}
		return layout, nil
	}
	if err != nil {
		return layout, fmt.Errorf("checking workspace symlink %s: %w", layout.LinkPath, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return layout, fmt.Errorf("workspace symlink path %s already exists and is not a symlink", layout.LinkPath)
	}

	target, err := os.Readlink(layout.LinkPath)
	if err != nil {
		return layout, fmt.Errorf("reading workspace symlink %s: %w", layout.LinkPath, err)
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(layout.LinkPath), target)
	}

	same, err := samePath(target, layout.ProjectRoot)
	if err != nil {
		return layout, err
	}
	if !same {
		return layout, fmt.Errorf("workspace symlink %s points to %s, want %s", layout.LinkPath, target, layout.ProjectRoot)
	}
	return layout, nil
}

func expandWorkspacePath(path, townRoot string) (string, error) {
	expanded := os.ExpandEnv(strings.TrimSpace(path))
	if strings.HasPrefix(expanded, "~/") || expanded == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolving home directory for %q: %w", path, err)
		}
		if expanded == "~" {
			expanded = home
		} else {
			expanded = filepath.Join(home, strings.TrimPrefix(expanded, "~/"))
		}
	}
	if !filepath.IsAbs(expanded) {
		expanded = filepath.Join(townRoot, expanded)
	}
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Errorf("resolving workspace root %q: %w", path, err)
	}
	return filepath.Clean(abs), nil
}

func validateSymlinkName(name string) error {
	if name == "" || name == "." || name == ".." {
		return fmt.Errorf("invalid workspace project_symlink %q", name)
	}
	if name != filepath.Base(name) || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("workspace project_symlink %q must be a single path segment", name)
	}
	return nil
}

func stablePathSegment(value, fallback string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(value) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return fallback
	}
	return out
}

func samePath(a, b string) (bool, error) {
	absA, err := filepath.Abs(a)
	if err != nil {
		return false, fmt.Errorf("resolving %s: %w", a, err)
	}
	absB, err := filepath.Abs(b)
	if err != nil {
		return false, fmt.Errorf("resolving %s: %w", b, err)
	}
	evalA, err := filepath.EvalSymlinks(absA)
	if err != nil {
		return false, fmt.Errorf("resolving symlinks for %s: %w", absA, err)
	}
	evalB, err := filepath.EvalSymlinks(absB)
	if err != nil {
		return false, fmt.Errorf("resolving symlinks for %s: %w", absB, err)
	}
	return filepath.Clean(evalA) == filepath.Clean(evalB), nil
}
