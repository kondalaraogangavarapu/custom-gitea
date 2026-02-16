package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Engine manages git repositories on disk.
type Engine struct {
	rootPath string
}

// NewEngine creates a git engine rooted at the given path.
func NewEngine(rootPath string) *Engine {
	return &Engine{rootPath: rootPath}
}

// RepoPath returns the on-disk path for a repository.
func (e *Engine) RepoPath(owner, name string) string {
	return filepath.Join(e.rootPath, owner, name+".git")
}

// InitRepo creates a new bare git repository.
func (e *Engine) InitRepo(owner, name, defaultBranch string) error {
	path := e.RepoPath(owner, name)
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	cmd := exec.Command("git", "init", "--bare", "--initial-branch="+defaultBranch, path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git init: %s: %w", string(out), err)
	}
	return nil
}

// DeleteRepo removes a repository from disk.
func (e *Engine) DeleteRepo(owner, name string) error {
	return os.RemoveAll(e.RepoPath(owner, name))
}

// ListBranches returns branch names for a repository.
func (e *Engine) ListBranches(owner, name string) ([]string, error) {
	path := e.RepoPath(owner, name)
	cmd := exec.Command("git", "-C", path, "branch", "--format=%(refname:short)")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var branches []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			branches = append(branches, l)
		}
	}
	return branches, nil
}

// CommitInfo represents basic commit metadata.
type CommitInfo struct {
	SHA     string    `json:"sha"`
	Message string    `json:"message"`
	Author  string    `json:"author"`
	Date    time.Time `json:"date"`
}

// Log returns recent commits for a branch.
func (e *Engine) Log(owner, name, branch string, limit int) ([]CommitInfo, error) {
	path := e.RepoPath(owner, name)
	format := "%H|%s|%an|%aI"
	cmd := exec.Command("git", "-C", path, "log", branch, fmt.Sprintf("--format=%s", format), fmt.Sprintf("-n%d", limit))
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var commits []CommitInfo
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}
		t, _ := time.Parse(time.RFC3339, parts[3])
		commits = append(commits, CommitInfo{
			SHA:     parts[0],
			Message: parts[1],
			Author:  parts[2],
			Date:    t,
		})
	}
	return commits, nil
}

// ListTree returns file/directory entries at a path in a given ref.
type TreeEntry struct {
	Mode string `json:"mode"`
	Type string `json:"type"` // blob, tree
	SHA  string `json:"sha"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

func (e *Engine) ListTree(owner, name, ref, path string) ([]TreeEntry, error) {
	repoPath := e.RepoPath(owner, name)
	treeish := ref
	if path != "" {
		treeish = ref + ":" + path
	}
	cmd := exec.Command("git", "-C", repoPath, "ls-tree", "-l", treeish)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var entries []TreeEntry
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		// format: mode type sha size\tname
		parts := strings.Fields(line)
		if len(parts) < 5 {
			continue
		}
		namePart := parts[4]
		if idx := strings.Index(line, "\t"); idx >= 0 {
			namePart = line[idx+1:]
		}
		var size int64
		fmt.Sscanf(parts[3], "%d", &size)
		entries = append(entries, TreeEntry{
			Mode: parts[0],
			Type: parts[1],
			SHA:  parts[2],
			Size: size,
			Name: namePart,
		})
	}
	return entries, nil
}

// ReadBlob returns the content of a file at a given ref and path.
func (e *Engine) ReadBlob(owner, name, ref, path string) ([]byte, error) {
	repoPath := e.RepoPath(owner, name)
	cmd := exec.Command("git", "-C", repoPath, "show", ref+":"+path)
	return cmd.Output()
}

// CreateBranch creates a new branch from a base ref.
func (e *Engine) CreateBranch(owner, name, branch, baseRef string) error {
	repoPath := e.RepoPath(owner, name)
	cmd := exec.Command("git", "-C", repoPath, "branch", branch, baseRef)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("create branch: %s: %w", string(out), err)
	}
	return nil
}
