package ops

import (
	"os"
	"path/filepath"
	"testing"
)

// ── parseTargetPath ─────────────────────────────────────────────────

func TestParseTargetPath(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "standard writeString pattern",
			content: `Files.writeString(Path.of("target/openapi.yml"), content);`,
			want:    "target/openapi.yml",
		},
		{
			name:    "with extra whitespace",
			content: `Files.writeString( Path.of( "target/generated/api.yaml" ), content);`,
			want:    "target/generated/api.yaml",
		},
		{
			name:    "no match without writeString",
			content: `Files.write(Path.of("target/openapi.yml"), bytes);`,
			want:    "",
		},
		{
			name:    "empty content",
			content: "",
			want:    "",
		},
		{
			name:    "embedded in multiline class",
			content: "class Test {\n  void run() {\n    Files.writeString(Path.of(\"target/openapi.yml\"), spec);\n  }\n}",
			want:    "target/openapi.yml",
		},
		{
			name:    "unrelated code",
			content: `System.out.println("hello");`,
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTargetPath(tt.content)
			if got != tt.want {
				t.Errorf("parseTargetPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ── parseResourcePath ───────────────────────────────────────────────

func TestParseResourcePath(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		setupDir string // relative dir inside repoDir to create the file in
		wantRel  string // expected relative path result
	}{
		{
			name:     "integration-test resources with leading slash",
			content:  `@Value("classpath:/api/openapi.yaml")`,
			setupDir: filepath.Join("src", "integration-test", "resources", "api"),
			wantRel:  filepath.Join("src", "integration-test", "resources", "api", "openapi.yaml"),
		},
		{
			name:     "main resources without leading slash",
			content:  `@Value("classpath:api/openapi.yaml")`,
			setupDir: filepath.Join("src", "main", "resources", "api"),
			wantRel:  filepath.Join("src", "main", "resources", "api", "openapi.yaml"),
		},
		{
			name:     "test resources",
			content:  `@Value("classpath:/openapi/spec.yml")`,
			setupDir: filepath.Join("src", "test", "resources", "openapi"),
			wantRel:  filepath.Join("src", "test", "resources", "openapi", "spec.yml"),
		},
		{
			name:     "no classpath annotation",
			content:  `String path = "api/openapi.yaml";`,
			setupDir: "",
			wantRel:  "",
		},
		{
			name:     "classpath but file missing on disk",
			content:  `@Value("classpath:/api/openapi.yaml")`,
			setupDir: "",
			wantRel:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoDir := t.TempDir()

			if tt.setupDir != "" {
				dir := filepath.Join(repoDir, tt.setupDir)
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				filename := filepath.Base(tt.wantRel)
				if err := os.WriteFile(filepath.Join(dir, filename), []byte("spec"), 0644); err != nil {
					t.Fatal(err)
				}
			}

			got := parseResourcePath(tt.content, repoDir)
			if got != tt.wantRel {
				t.Errorf("parseResourcePath() = %q, want %q", got, tt.wantRel)
			}
		})
	}
}

func TestParseResourcePathPriority(t *testing.T) {
	repoDir := t.TempDir()
	content := `@Value("classpath:/api/openapi.yaml")`

	// Create the file in both integration-test and main resources.
	// integration-test should win because it's checked first.
	for _, root := range []string{
		filepath.Join("src", "integration-test", "resources", "api"),
		filepath.Join("src", "main", "resources", "api"),
	} {
		dir := filepath.Join(repoDir, root)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "openapi.yaml"), []byte("spec"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	got := parseResourcePath(content, repoDir)
	want := filepath.Join("src", "integration-test", "resources", "api", "openapi.yaml")
	if got != want {
		t.Errorf("parseResourcePath() = %q, want %q (should prefer integration-test)", got, want)
	}
}

// ── OwnSpecCurrent ──────────────────────────────────────────────────

func TestOwnSpecCurrent(t *testing.T) {
	tests := []struct {
		name       string
		pomVersion string
		specVer    string
		want       bool
	}{
		{
			name:       "versions match",
			pomVersion: "2.0.0",
			specVer:    "2.0.0",
			want:       true,
		},
		{
			name:       "versions differ",
			pomVersion: "3.0.0",
			specVer:    "2.0.0",
			want:       false,
		},
		{
			name:       "empty pom version",
			pomVersion: "",
			specVer:    "1.2.3",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			pomXML := `<?xml version="1.0" encoding="UTF-8"?>
<project>
  <parent>
    <groupId>se.sundsvall.dept44</groupId>
    <artifactId>dept44-service-parent</artifactId>
    <version>5.0.0</version>
  </parent>
  <version>` + tt.pomVersion + `</version>
</project>`
			writePom(t, dir, pomXML)

			if tt.specVer != "" {
				specDir := filepath.Join(dir, "src", "integration-test", "resources")
				if err := os.MkdirAll(specDir, 0755); err != nil {
					t.Fatal(err)
				}
				specContent := "openapi: 3.0.1\ninfo:\n  title: Test API\n  version: " + tt.specVer + "\n"
				if err := os.WriteFile(filepath.Join(specDir, "openapi.yaml"), []byte(specContent), 0644); err != nil {
					t.Fatal(err)
				}
			}

			got := OwnSpecCurrent(dir)
			if got != tt.want {
				t.Errorf("OwnSpecCurrent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOwnSpecCurrentNoPom(t *testing.T) {
	dir := t.TempDir()
	if OwnSpecCurrent(dir) {
		t.Error("OwnSpecCurrent() should return false when no pom.xml exists")
	}
}

func TestOwnSpecCurrentNoSpec(t *testing.T) {
	dir := t.TempDir()
	writePom(t, dir, `<?xml version="1.0"?><project><version>1.0.0</version></project>`)
	if OwnSpecCurrent(dir) {
		t.Error("OwnSpecCurrent() should return false when no spec file exists")
	}
}

// ── CopySpec ────────────────────────────────────────────────────────

func TestCopySpec(t *testing.T) {
	dir := t.TempDir()

	srcContent := "openapi: 3.0.1\ninfo:\n  title: Generated\n  version: 2.0.0\n"
	dstContent := "openapi: 3.0.1\ninfo:\n  title: Old\n  version: 1.0.0\n"

	paths := SpecPaths{
		TargetFile:   "target/openapi.yml",
		ResourceFile: filepath.Join("src", "integration-test", "resources", "openapi.yaml"),
	}

	// Create source file
	srcDir := filepath.Join(dir, "target")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "openapi.yml"), []byte(srcContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create destination file with old content
	dstDir := filepath.Join(dir, "src", "integration-test", "resources")
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dstDir, "openapi.yaml"), []byte(dstContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := CopySpec(dir, paths); err != nil {
		t.Fatalf("CopySpec() error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dstDir, "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != srcContent {
		t.Errorf("CopySpec() destination = %q, want %q", string(got), srcContent)
	}
}

func TestCopySpecMissingSrc(t *testing.T) {
	dir := t.TempDir()
	paths := SpecPaths{
		TargetFile:   "target/nonexistent.yml",
		ResourceFile: "src/resources/openapi.yaml",
	}
	if err := CopySpec(dir, paths); err == nil {
		t.Error("CopySpec() expected error for missing source file")
	}
}

// ── findSpecTestFile ────────────────────────────────────────────────

func TestFindSpecTestFile(t *testing.T) {
	dir := t.TempDir()

	javaDir := filepath.Join(dir, "src", "integration-test", "java", "se", "sundsvall", "myservice")
	if err := os.MkdirAll(javaDir, 0755); err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(javaDir, "OpenApiSpecificationIT.java")
	if err := os.WriteFile(testFile, []byte("class OpenApiSpecificationIT {}"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := findSpecTestFile(dir)
	if err != nil {
		t.Fatalf("findSpecTestFile() error: %v", err)
	}
	if got != testFile {
		t.Errorf("findSpecTestFile() = %q, want %q", got, testFile)
	}
}

func TestFindSpecTestFileNotFound(t *testing.T) {
	dir := t.TempDir()

	// Create the parent dir but no Java file inside
	itDir := filepath.Join(dir, "src", "integration-test", "java")
	if err := os.MkdirAll(itDir, 0755); err != nil {
		t.Fatal(err)
	}

	_, err := findSpecTestFile(dir)
	if err == nil {
		t.Error("findSpecTestFile() expected error when file not found")
	}
}

func TestFindSpecTestFileMissingDir(t *testing.T) {
	dir := t.TempDir()
	_, err := findSpecTestFile(dir)
	if err == nil {
		t.Error("findSpecTestFile() expected error when integration-test dir missing")
	}
}

// ── ParseSpecPaths (full orchestration) ─────────────────────────────

func TestParseSpecPaths(t *testing.T) {
	dir := t.TempDir()

	// Create the Java test file with both patterns
	javaDir := filepath.Join(dir, "src", "integration-test", "java", "se", "sundsvall", "myservice")
	if err := os.MkdirAll(javaDir, 0755); err != nil {
		t.Fatal(err)
	}

	javaContent := `package se.sundsvall.myservice;

import java.nio.file.Files;
import java.nio.file.Path;
import org.springframework.beans.factory.annotation.Value;

class OpenApiSpecificationIT {

    @Value("classpath:/api/openapi.yaml")
    private String specPath;

    void generateSpec() throws Exception {
        String spec = getSpec();
        Files.writeString(Path.of("target/openapi.yml"), spec);
    }
}
`
	if err := os.WriteFile(filepath.Join(javaDir, "OpenApiSpecificationIT.java"), []byte(javaContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create the resource file that the classpath resolves to
	resDir := filepath.Join(dir, "src", "integration-test", "resources", "api")
	if err := os.MkdirAll(resDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resDir, "openapi.yaml"), []byte("openapi: 3.0.1"), 0644); err != nil {
		t.Fatal(err)
	}

	paths, err := ParseSpecPaths(dir)
	if err != nil {
		t.Fatalf("ParseSpecPaths() error: %v", err)
	}

	if paths.TargetFile != "target/openapi.yml" {
		t.Errorf("TargetFile = %q, want %q", paths.TargetFile, "target/openapi.yml")
	}

	wantResource := filepath.Join("src", "integration-test", "resources", "api", "openapi.yaml")
	if paths.ResourceFile != wantResource {
		t.Errorf("ResourceFile = %q, want %q", paths.ResourceFile, wantResource)
	}
}

func TestParseSpecPathsNoTargetPattern(t *testing.T) {
	dir := t.TempDir()

	javaDir := filepath.Join(dir, "src", "integration-test", "java", "se", "sundsvall")
	if err := os.MkdirAll(javaDir, 0755); err != nil {
		t.Fatal(err)
	}

	javaContent := `class OpenApiSpecificationIT {
    @Value("classpath:/api/openapi.yaml")
    private String specPath;
}
`
	if err := os.WriteFile(filepath.Join(javaDir, "OpenApiSpecificationIT.java"), []byte(javaContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ParseSpecPaths(dir)
	if err == nil {
		t.Error("ParseSpecPaths() expected error when writeString target pattern is missing")
	}
}

func TestParseSpecPathsNoClasspathResource(t *testing.T) {
	dir := t.TempDir()

	javaDir := filepath.Join(dir, "src", "integration-test", "java", "se", "sundsvall")
	if err := os.MkdirAll(javaDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Has writeString but no @Value classpath and no resource file on disk
	javaContent := `class OpenApiSpecificationIT {
    void run() throws Exception {
        Files.writeString(Path.of("target/openapi.yml"), spec);
    }
}
`
	if err := os.WriteFile(filepath.Join(javaDir, "OpenApiSpecificationIT.java"), []byte(javaContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ParseSpecPaths(dir)
	if err == nil {
		t.Error("ParseSpecPaths() expected error when classpath resource is missing")
	}
}

func TestParseSpecPathsNoTestFile(t *testing.T) {
	dir := t.TempDir()
	_, err := ParseSpecPaths(dir)
	if err == nil {
		t.Error("ParseSpecPaths() expected error when no test file exists")
	}
}

// ── TestGroupStaleByRepo (kept from existing tests) ─────────────────

// ── CopySpec write error ────────────────────────────────────────────

func TestCopySpecWriteError(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "target")
	os.MkdirAll(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "openapi.yml"), []byte("content"), 0644)

	paths := SpecPaths{
		TargetFile:   "target/openapi.yml",
		ResourceFile: filepath.Join("nonexistent", "subdir", "openapi.yaml"),
	}

	err := CopySpec(dir, paths)
	if err == nil {
		t.Error("expected error for write to nonexistent directory")
	}
}

// ── ParseSpecPaths read error ───────────────────────────────────────

func TestParseSpecPathsReadError(t *testing.T) {
	dir := t.TempDir()
	// Create the file location but make it unreadable
	itDir := filepath.Join(dir, "src", "integration-test", "java", "se", "sundsvall")
	os.MkdirAll(itDir, 0755)
	testFile := filepath.Join(itDir, "OpenApiSpecificationIT.java")
	os.WriteFile(testFile, []byte("content"), 0644)
	os.Chmod(testFile, 0000)
	t.Cleanup(func() { os.Chmod(testFile, 0644) })

	_, err := ParseSpecPaths(dir)
	if err == nil {
		t.Error("expected error for unreadable test file")
	}
}

func TestGroupStaleByRepo(t *testing.T) {
	staleList := []StaleIntegration{
		{Repo: "alpha", TargetName: "svc-a"},
		{Repo: "beta", TargetName: "svc-b"},
		{Repo: "alpha", TargetName: "svc-c"},
	}

	byRepo, order := groupStaleByRepo(staleList)

	if len(order) != 2 {
		t.Fatalf("got %d repos, want 2", len(order))
	}
	if order[0] != "alpha" || order[1] != "beta" {
		t.Errorf("order = %v, want [alpha, beta]", order)
	}
	if len(byRepo["alpha"]) != 2 {
		t.Errorf("alpha has %d entries, want 2", len(byRepo["alpha"]))
	}
	if len(byRepo["beta"]) != 1 {
		t.Errorf("beta has %d entries, want 1", len(byRepo["beta"]))
	}
}
