package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type executionTarget struct {
	ProjectPath string
	WorkDir     string
	BaseBranch  string
	Isolated    bool
}

func prepareExecutionTarget(ctx context.Context, projectPath string) (executionTarget, error) {
	abs, err := filepath.Abs(projectPath)
	if err != nil {
		return executionTarget{}, fmt.Errorf("abs project path: %w", err)
	}

	branch, _ := orchestratorCurrentBranch(ctx, abs)
	target := executionTarget{ProjectPath: abs, WorkDir: abs, BaseBranch: branch}

	dirty, err := gitDirty(ctx, abs)
	if err != nil || !dirty {
		return target, nil
	}

	hasOriginMain, err := gitHasOriginMain(ctx, abs)
	if err != nil || !hasOriginMain {
		return target, nil
	}

	if err := gitFetchOriginMain(ctx, abs); err != nil {
		return target, nil
	}

	root := filepath.Join(abs, ".nightshift", "worktrees")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return executionTarget{}, fmt.Errorf("create worktree root: %w", err)
	}

	name := fmt.Sprintf("run-%s", time.Now().Format("20060102-150405"))
	workDir := filepath.Join(root, name)
	cmd := exec.CommandContext(ctx, "git", "-C", abs, "worktree", "add", "--detach", workDir, "origin/main")
	if out, err := cmd.CombinedOutput(); err != nil {
		return executionTarget{}, fmt.Errorf("git worktree add: %w: %s", err, strings.TrimSpace(string(out)))
	}

	target.WorkDir = workDir
	target.BaseBranch = ""
	target.Isolated = true
	return target, nil
}

func gitDirty(ctx context.Context, repo string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repo, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) != "", nil
}

func gitHasOriginMain(ctx context.Context, repo string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repo, "rev-parse", "--verify", "refs/remotes/origin/main")
	if err := cmd.Run(); err == nil {
		return true, nil
	}
	cmd = exec.CommandContext(ctx, "git", "-C", repo, "rev-parse", "--verify", "refs/remotes/origin/master")
	if err := cmd.Run(); err == nil {
		return true, nil
	}
	return false, nil
}

func gitFetchOriginMain(ctx context.Context, repo string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repo, "fetch", "origin", "main")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("fetch origin main: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func orchestratorCurrentBranch(ctx context.Context, repo string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repo, "branch", "--show-current")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
