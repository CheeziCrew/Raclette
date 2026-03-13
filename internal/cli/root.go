package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"

	"github.com/CheeziCrew/raclette/internal/ops"
)

var jsonOutput bool

var (
	ok   = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("✔")
	fail = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("✗")
	dim  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// BuildCLI creates the Cobra command tree for raclette CLI usage.
func BuildCLI() *cobra.Command {
	root := &cobra.Command{
		Use:           "raclette",
		Short:         "Raclette — Maven multi-repo operations for dept44",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output JSON instead of styled text")

	root.AddCommand(formatCmd())
	root.AddCommand(verifyCmd())
	root.AddCommand(findDepCmd())
	root.AddCommand(findStaleCmd())
	root.AddCommand(replaceDepCmd())
	root.AddCommand(swapDepCmd())
	root.AddCommand(bumpParentCmd())
	root.AddCommand(updateSpecsCmd())
	root.AddCommand(updateIntegrationSpecsCmd())

	return root
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// discoverRepos finds all dept44 maven repos in the current directory.
func discoverRepos() []string {
	base, _ := os.Getwd()
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil
	}

	var dirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		repoPath := filepath.Join(base, e.Name())
		if _, err := os.Stat(filepath.Join(repoPath, ".git")); err != nil {
			continue
		}
		if _, err := os.Stat(filepath.Join(repoPath, "pom.xml")); err != nil {
			continue
		}
		if !ops.IsDept44Repo(repoPath) {
			continue
		}
		dirs = append(dirs, repoPath)
	}
	return dirs
}

// ── Maven commands ──────────────────────────────────────────────────

func formatCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "format",
		Short: "Apply dept44 formatting across repos",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMavenParallel(discoverRepos(), []string{"clean", "dept44-formatting:apply"})
		},
		Args: cobra.NoArgs,
	}
}

func verifyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "verify",
		Short: "Format + verify across repos",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMavenParallel(discoverRepos(), []string{"clean", "dept44-formatting:apply", "verify"})
		},
		Args: cobra.NoArgs,
	}
}

type mavenResult struct {
	Repo    string `json:"repo"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func runMavenParallel(dirs []string, mvnArgs []string) error {
	if len(dirs) == 0 {
		fmt.Println("No dept44 repos found.")
		return nil
	}

	var mu sync.Mutex
	var results []mavenResult
	var wg sync.WaitGroup
	sem := make(chan struct{}, 3)

	for _, dir := range dirs {
		wg.Add(1)
		sem <- struct{}{}
		go func(d string) {
			defer wg.Done()
			defer func() { <-sem }()
			repo := filepath.Base(d)
			errMsg := runMvn(mvnArgs, d)
			mu.Lock()
			if errMsg != "" {
				results = append(results, mavenResult{Repo: repo, Error: errMsg})
			} else {
				results = append(results, mavenResult{Repo: repo, Success: true})
			}
			mu.Unlock()
		}(dir)
	}
	wg.Wait()

	if jsonOutput {
		return printJSON(results)
	}
	for _, r := range results {
		if r.Success {
			fmt.Printf(" %s %s\n", ok, r.Repo)
		} else {
			fmt.Printf(" %s %s  %s\n", fail, r.Repo, dim.Render(r.Error))
		}
	}
	return nil
}

// runMvnFunc allows tests to mock the mvn runner.
var runMvnFunc = defaultRunMvn

func runMvn(args []string, dir string) string {
	return runMvnFunc(args, dir)
}

// defaultRunMvn runs mvn and returns error message or "".
func defaultRunMvn(args []string, dir string) string {
	argsCopy := make([]string, len(args))
	copy(argsCopy, args)
	argsCopy = append([]string{"-B"}, argsCopy...)

	cmd := exec.Command("mvn", argsCopy...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Parse first [ERROR] line.
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "[ERROR]") {
				cleaned := strings.TrimSpace(strings.TrimPrefix(line, "[ERROR]"))
				if cleaned == "" || strings.HasPrefix(cleaned, "[Help") || strings.HasPrefix(cleaned, "-> [Help") || strings.HasPrefix(cleaned, "http") {
					continue
				}
				return cleaned
			}
		}
		return err.Error()
	}
	return ""
}

// ── Scan commands ───────────────────────────────────────────────────

func findDepCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "find-dep <query>",
		Short: "Find repos using a specific dependency",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			matches := ops.FindDependency(discoverRepos(), args[0])
			if jsonOutput {
				return printJSON(matches)
			}
			if len(matches) == 0 {
				fmt.Println("No matches found.")
				return nil
			}
			for _, m := range matches {
				line := fmt.Sprintf(" %s  %s:%s", m.Repo, m.GroupID, m.Artifact)
				if m.Version != "" {
					line += "  " + m.Version
				}
				if m.InParent {
					line += "  [parent]"
				}
				fmt.Println(line)
			}
			return nil
		},
	}
}

func findStaleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "find-stale",
		Short: "Find repos with outdated OpenAPI integration specs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			stale := ops.FindStaleSpecs(discoverRepos())
			if jsonOutput {
				return printJSON(stale)
			}
			if len(stale) == 0 {
				fmt.Println("All integration specs are up to date.")
				return nil
			}
			for _, s := range stale {
				fmt.Printf(" %-35s %s  %s → %s\n", s.Repo, s.SpecFile, s.SpecVersion, s.TargetVersion)
			}
			return nil
		},
	}
}

// ── Transform commands ──────────────────────────────────────────────

func replaceDepCmd() *cobra.Command {
	var groupID, artifactID, version string
	cmd := &cobra.Command{
		Use:   "replace-dep",
		Short: "Replace a dependency version across repos",
		RunE: func(cmd *cobra.Command, args []string) error {
			results := ops.ReplaceDependencyVersion(discoverRepos(), groupID, artifactID, version)
			return printTransformResults(results)
		},
	}
	cmd.Flags().StringVarP(&groupID, "group", "g", "", "Group ID (required)")
	cmd.Flags().StringVarP(&artifactID, "artifact", "a", "", "Artifact ID (required)")
	cmd.Flags().StringVarP(&version, "version", "v", "", "New version (required)")
	cmd.MarkFlagRequired("group")
	cmd.MarkFlagRequired("artifact")
	cmd.MarkFlagRequired("version")
	return cmd
}

func swapDepCmd() *cobra.Command {
	var oldGroup, oldArtifact, newGroup, newArtifact, version string
	cmd := &cobra.Command{
		Use:   "swap-dep",
		Short: "Replace one dependency with another across repos",
		RunE: func(cmd *cobra.Command, args []string) error {
			results := ops.SwapDependency(discoverRepos(), oldGroup, oldArtifact, newGroup, newArtifact, version)
			return printTransformResults(results)
		},
	}
	cmd.Flags().StringVar(&oldGroup, "old-group", "", "Old group ID (required)")
	cmd.Flags().StringVar(&oldArtifact, "old-artifact", "", "Old artifact ID (required)")
	cmd.Flags().StringVar(&newGroup, "new-group", "", "New group ID (required)")
	cmd.Flags().StringVar(&newArtifact, "new-artifact", "", "New artifact ID (required)")
	cmd.Flags().StringVarP(&version, "version", "v", "", "New version (required)")
	cmd.MarkFlagRequired("old-group")
	cmd.MarkFlagRequired("old-artifact")
	cmd.MarkFlagRequired("new-group")
	cmd.MarkFlagRequired("new-artifact")
	cmd.MarkFlagRequired("version")
	return cmd
}

func bumpParentCmd() *cobra.Command {
	var version string
	cmd := &cobra.Command{
		Use:   "bump-parent",
		Short: "Update dept44-parent version across repos",
		RunE: func(cmd *cobra.Command, args []string) error {
			results := ops.BumpParentVersion(discoverRepos(), version)
			return printTransformResults(results)
		},
	}
	cmd.Flags().StringVarP(&version, "version", "v", "", "New parent version (required)")
	cmd.MarkFlagRequired("version")
	return cmd
}

func updateSpecsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update-specs",
		Short: "Update own OpenAPI specs via integration test",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMavenParallel(discoverRepos(), []string{
				"verify",
				"-Dit.test=OpenApiSpecificationIT",
				"-Dfailsafe.failIfNoSpecifiedTests=false",
				"-Dsurefire.skip",
				"-Dspotless.skip=true",
				"-Dcheckstyle.skip",
				"-Denforcer.skip",
				"-Djacoco.skip=true",
			})
		},
	}
}

func updateIntegrationSpecsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update-integration-specs",
		Short: "Pull fresh specs from target services into your integrations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			results := ops.UpdateIntegrationSpecs(discoverRepos())
			return printTransformResults(results)
		},
	}
}

func printTransformResults(results []ops.Result) error {
	if jsonOutput {
		return printJSON(results)
	}
	for _, r := range results {
		switch {
		case r.Err != nil:
			fmt.Printf(" %s %-35s %v\n", fail, r.Repo, r.Err)
		case r.Changed:
			fmt.Printf(" %s %-35s %s\n", ok, r.Repo, r.Message)
		case r.Repo != "":
			fmt.Printf("   %-35s %s\n", r.Repo, dim.Render(r.Message))
		default:
			fmt.Printf("   %s\n", r.Message)
		}
	}
	return nil
}
