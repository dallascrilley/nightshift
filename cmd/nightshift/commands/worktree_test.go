package commands

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareExecutionTarget_CleanRepoUsesOriginalPath(t *testing.T) {
	repo := initRepoWithOriginMain(t)

	target, err := prepareExecutionTarget(context.Background(), repo)
	if err != nil {
		t.Fatalf("prepareExecutionTarget: %v", err)
	}
	if target.WorkDir != repo {
		t.Fatalf("WorkDir = %q, want %q", target.WorkDir, repo)
	}
	if target.Isolated {
		t.Fatal("expected non-isolated target for clean repo")
	}
	if target.BaseBranch != "main" {
		t.Fatalf("BaseBranch = %q, want main", target.BaseBranch)
	}
}

func TestPrepareExecutionTarget_DirtyRepoUsesOriginMainWorktree(t *testing.T) {
	repo := initRepoWithOriginMain(t)
	readme := filepath.Join(repo, "README.md")
	if err := os.WriteFile(readme, []byte("dirty local change\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "scratch.txt"), []byte("untracked\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	target, err := prepareExecutionTarget(context.Background(), repo)
	if err != nil {
		t.Fatalf("prepareExecutionTarget: %v", err)
	}
	if !target.Isolated {
		t.Fatal("expected isolated target for dirty repo")
	}
	if target.WorkDir == repo {
		t.Fatal("expected worktree path to differ from original repo")
	}
	if target.BaseBranch != "" {
		t.Fatalf("BaseBranch = %q, want empty for isolated worktree", target.BaseBranch)
	}

	content, err := os.ReadFile(filepath.Join(target.WorkDir, "README.md"))
	if err != nil {
		t.Fatalf("read worktree README: %v", err)
	}
	if strings.Contains(string(content), "dirty local change") {
		t.Fatalf("worktree README contains dirty local changes: %q", string(content))
	}
	if _, err := os.Stat(filepath.Join(target.WorkDir, "scratch.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected untracked scratch.txt to be absent in isolated worktree, got err=%v", err)
	}
}

func initRepoWithOriginMain(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	remote := filepath.Join(base, "remote.git")
	repo := filepath.Join(base, "repo")
	runGit(t, base, "init", "--bare", remote)
	runGit(t, base, "clone", remote, repo)
	runGit(t, repo, "config", "user.name", "Nightshift Test")
	runGit(t, repo, "config", "user.email", "nightshift@example.com")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("hello from main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-m", "feat: initial commit")
	runGit(t, repo, "branch", "-M", "main")
	runGit(t, repo, "push", "-u", "origin", "main")
	return repo
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
	return string(out)
}
