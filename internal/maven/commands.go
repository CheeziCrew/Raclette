package maven

// Kind describes how a command executes.
type Kind int

const (
	KindMaven      Kind = iota // Runs mvn with Args across selected repos
	KindScan                   // Reads files, produces a report (no mutations)
	KindTransform              // Modifies files across repos (pom.xml, specs, etc.)
	KindUpdateSpec             // Run OpenAPI spec test → copy generated spec → run again
)

// Command represents an operation with metadata for the TUI.
type Command struct {
	Name        string
	Description string
	Icon        string
	Kind        Kind

	// KindMaven fields
	Args []string

	// Input prompts — the TUI should collect these before running.
	// e.g. "groupId", "artifactId", "version"
	Prompts []Prompt
}

// Prompt describes a user input the command needs before executing.
type Prompt struct {
	Key         string // internal key, e.g. "groupId"
	Label       string // display label, e.g. "Group ID"
	Placeholder string // hint text
}

// Commands returns all available raclette commands.
func Commands() []Command {
	return []Command{
		// ── Maven (multi-repo) ──────────────────────────────────────
		{
			Name:        "Format All",
			Description: "Apply dept44 formatting across repos",
			Icon:        "✨",
			Kind:        KindMaven,
			Args:        []string{"clean", "dept44-formatting:apply"},
		},
		{
			Name:        "Verify All",
			Description: "Format + verify across repos",
			Icon:        "🚀",
			Kind:        KindMaven,
			Args:        []string{"clean", "dept44-formatting:apply", "verify"},
		},
		{
			Name:        "Update Own Specs",
			Description: "Regenerate OpenAPI spec and commit it back to resources",
			Icon:        "🔌",
			Kind:        KindUpdateSpec,
			Args:        []string{"verify", "-Dit.test=OpenApiSpecificationIT", "-Dfailsafe.failIfNoSpecifiedTests=false", "-Dsurefire.skip", "-Dspotless.skip=true", "-Dcheckstyle.skip", "-Denforcer.skip", "-Djacoco.skip=true"},
		},

		// ── Scan (read-only) ────────────────────────────────────────
		{
			Name:        "Find Dependency",
			Description: "Which repos use a specific dependency?",
			Icon:        "🔍",
			Kind:        KindScan,
			Prompts: []Prompt{
				{Key: "query", Label: "Dependency", Placeholder: "e.g. lombok, jackson-databind, dept44-parent"},
			},
		},
		{
			Name:        "Find Stale Specs",
			Description: "List repos with outdated OpenAPI integration specs",
			Icon:        "👃",
			Kind:        KindScan,
		},

		// ── Transform (mutates files) ───────────────────────────────
		{
			Name:        "Replace Dependency Version",
			Description: "Bump a dependency version across repos",
			Icon:        "📦",
			Kind:        KindTransform,
			Prompts: []Prompt{
				{Key: "groupId", Label: "Group ID", Placeholder: "e.g. org.projectlombok"},
				{Key: "artifactId", Label: "Artifact ID", Placeholder: "e.g. lombok"},
				{Key: "newVersion", Label: "New Version", Placeholder: "e.g. 1.18.32"},
			},
		},
		{
			Name:        "Swap Dependency",
			Description: "Replace one dependency with another across repos",
			Icon:        "🔄",
			Kind:        KindTransform,
			Prompts: []Prompt{
				{Key: "oldGroupId", Label: "Old Group ID", Placeholder: "e.g. org.zalando"},
				{Key: "oldArtifactId", Label: "Old Artifact ID", Placeholder: "e.g. problem-spring-web"},
				{Key: "newGroupId", Label: "New Group ID", Placeholder: "e.g. se.sundsvall.dept44"},
				{Key: "newArtifactId", Label: "New Artifact ID", Placeholder: "e.g. dept44-problem"},
				{Key: "newVersion", Label: "New Version", Placeholder: "e.g. 1.0.0"},
			},
		},
		{
			Name:        "Bump Parent Version",
			Description: "Update dept44-parent version across repos",
			Icon:        "⬆️",
			Kind:        KindTransform,
			Prompts: []Prompt{
				{Key: "newVersion", Label: "New Parent Version", Placeholder: "e.g. 8.0.2"},
			},
		},
		{
			Name:        "Update Integration Specs",
			Description: "Pull fresh specs from target services into your integrations",
			Icon:        "🔗",
			Kind:        KindTransform,
		},
	}
}
