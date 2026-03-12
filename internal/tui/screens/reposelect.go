package screens

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/CheeziCrew/raclette/internal/ops"
	"github.com/CheeziCrew/curd"
)

// Re-export for backward compat.
type RepoSelectModel = curd.RepoSelectModel

// NewRepoSelect creates a repo selector using raclette's maven scanner.
func NewRepoSelect(rootPath string, termHeight, parentOffset int) curd.RepoSelectModel {
	return curd.NewRepoSelectModel(curd.RepoSelectConfig{
		Palette:      curd.RaclettePalette,
		RootPath:     rootPath,
		TermHeight:   termHeight,
		ParentOffset: parentOffset,
		Scanner:      mavenScan,
	})
}

// mavenScan discovers dept44 maven repos.
func mavenScan(rootPath string) ([]curd.RepoInfo, error) {
	entries, err := os.ReadDir(rootPath)
	if err != nil {
		return nil, err
	}

	var repos []curd.RepoInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		repoPath := filepath.Join(rootPath, e.Name())
		gitPath := filepath.Join(repoPath, ".git")
		if _, err := os.Stat(gitPath); err != nil {
			continue
		}
		pomPath := filepath.Join(repoPath, "pom.xml")
		if _, err := os.Stat(pomPath); err != nil {
			continue
		}
		if !ops.IsDept44Repo(repoPath) {
			continue
		}

		name := e.Name()
		changes := countChanges(repoPath)
		defaultBranch := detectDefaultBranch(repoPath)
		branch := currentBranch(repoPath)

		totalChanges := changes.Modified + changes.Added + changes.Deleted + changes.Untracked
		isDirty := totalChanges > 0 || (branch != "" && branch != defaultBranch)

		repos = append(repos, curd.RepoInfo{
			Path:          repoPath,
			Name:          name,
			Branch:        branch,
			DefaultBranch: defaultBranch,
			Modified:      changes.Modified,
			Added:         changes.Added,
			Deleted:       changes.Deleted,
			Untracked:     changes.Untracked,
			IsDirty:       isDirty,
		})
	}

	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Name < repos[j].Name
	})

	return repos, nil
}

type gitChanges struct {
	Modified, Added, Deleted, Untracked int
}

func countChanges(repoPath string) gitChanges {
	cmd := exec.Command("git", "-C", repoPath, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return gitChanges{}
	}
	var c gitChanges
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if len(line) < 2 {
			continue
		}
		switch line[0:2] {
		case "??":
			c.Untracked++
		case " M", "MM", "AM":
			c.Modified++
		case "A ", "R ", "C ":
			c.Added++
		case " D", "D ":
			c.Deleted++
		default:
			c.Modified++
		}
	}
	return c
}

func currentBranch(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "branch", "--show-current")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func detectDefaultBranch(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "symbolic-ref", "refs/remotes/origin/HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "main"
	}
	ref := strings.TrimSpace(string(out))
	parts := strings.Split(ref, "/")
	return parts[len(parts)-1]
}
