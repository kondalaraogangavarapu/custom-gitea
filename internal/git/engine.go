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

// WriteFile commits a file into a bare repository using git plumbing commands.
// It writes the blob, creates a tree, and commits it onto the given branch.
func (e *Engine) WriteFile(owner, name, branch, filePath, content, commitMsg, authorName, authorEmail string) (string, error) {
	repoPath := e.RepoPath(owner, name)

	// 1. Write blob object
	hashObj := exec.Command("git", "-C", repoPath, "hash-object", "-w", "--stdin")
	hashObj.Stdin = strings.NewReader(content)
	blobSHA, err := hashObj.Output()
	if err != nil {
		return "", fmt.Errorf("hash-object: %w", err)
	}
	blobRef := strings.TrimSpace(string(blobSHA))

	// 2. Read existing tree (if branch exists), otherwise start empty
	var treeContent string
	existingTree := exec.Command("git", "-C", repoPath, "ls-tree", branch)
	if out, err := existingTree.Output(); err == nil {
		// Filter out the old entry for this file path, keep everything else
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if line == "" {
				continue
			}
			// Each line: "mode type sha\tname"
			if idx := strings.Index(line, "\t"); idx >= 0 {
				entryName := line[idx+1:]
				if entryName == filePath {
					continue // skip old version of this file
				}
			}
			treeContent += line + "\n"
		}
	}

	// Add the new/updated file entry
	treeContent += fmt.Sprintf("100644 blob %s\t%s\n", blobRef, filePath)

	// 3. Create tree object
	mkTree := exec.Command("git", "-C", repoPath, "mktree")
	mkTree.Stdin = strings.NewReader(treeContent)
	treeSHA, err := mkTree.Output()
	if err != nil {
		return "", fmt.Errorf("mktree: %w", err)
	}
	treeRef := strings.TrimSpace(string(treeSHA))

	// 4. Create commit object
	env := []string{
		"GIT_AUTHOR_NAME=" + authorName,
		"GIT_AUTHOR_EMAIL=" + authorEmail,
		"GIT_COMMITTER_NAME=" + authorName,
		"GIT_COMMITTER_EMAIL=" + authorEmail,
	}

	commitArgs := []string{"-C", repoPath, "commit-tree", treeRef, "-m", commitMsg}
	// If branch exists, set it as parent
	parentSHA := exec.Command("git", "-C", repoPath, "rev-parse", "--verify", branch)
	if parentOut, err := parentSHA.Output(); err == nil {
		commitArgs = append(commitArgs, "-p", strings.TrimSpace(string(parentOut)))
	}

	commitCmd := exec.Command("git", commitArgs...)
	commitCmd.Env = append(os.Environ(), env...)
	commitSHA, err := commitCmd.Output()
	if err != nil {
		return "", fmt.Errorf("commit-tree: %w", err)
	}
	commitRef := strings.TrimSpace(string(commitSHA))

	// 5. Update branch ref
	updateRef := exec.Command("git", "-C", repoPath, "update-ref", "refs/heads/"+branch, commitRef)
	if out, err := updateRef.CombinedOutput(); err != nil {
		return "", fmt.Errorf("update-ref: %s: %w", string(out), err)
	}

	return commitRef, nil
}
