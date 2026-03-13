package ops

import (
	"os"
	"path/filepath"
	"testing"
)

func writePom(t *testing.T, dir, content string) {
	t.Helper()
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(content), 0644)
}

const validPom = `<?xml version="1.0" encoding="UTF-8"?>
<project>
  <parent>
    <groupId>se.sundsvall.dept44</groupId>
    <artifactId>dept44-service-parent</artifactId>
    <version>5.0.0</version>
  </parent>
  <version>1.2.3</version>
  <properties>
    <spring.version>6.0.0</spring.version>
  </properties>
  <dependencies>
    <dependency>
      <groupId>org.springframework</groupId>
      <artifactId>spring-core</artifactId>
      <version>${spring.version}</version>
    </dependency>
  </dependencies>
</project>`

// ── TestReadPom ─────────────────────────────────────────────────────

func TestReadPom(t *testing.T) {
	t.Run("valid pom", func(t *testing.T) {
		dir := t.TempDir()
		writePom(t, dir, validPom)

		pom, data, err := ReadPom(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pom == nil {
			t.Fatal("pom is nil")
		}
		if data == nil {
			t.Fatal("data is nil")
		}
		if pom.Version != "1.2.3" {
			t.Errorf("version = %q, want 1.2.3", pom.Version)
		}
		if pom.Parent.ArtifactID != "dept44-service-parent" {
			t.Errorf("parent artifactId = %q", pom.Parent.ArtifactID)
		}
	})

	t.Run("invalid XML", func(t *testing.T) {
		dir := t.TempDir()
		writePom(t, dir, "<<<not xml>>>")

		_, _, err := ReadPom(dir)
		if err == nil {
			t.Fatal("expected error for invalid XML")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		dir := t.TempDir()

		_, _, err := ReadPom(dir)
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})
}

// ── TestIsDept44Repo ────────────────────────────────────────────────

func TestIsDept44Repo(t *testing.T) {
	t.Run("dept44 parent", func(t *testing.T) {
		dir := t.TempDir()
		writePom(t, dir, validPom)

		if !IsDept44Repo(dir) {
			t.Error("expected true for dept44 parent")
		}
	})

	t.Run("other parent", func(t *testing.T) {
		dir := t.TempDir()
		writePom(t, dir, `<project>
  <parent>
    <groupId>com.example</groupId>
    <artifactId>other-parent</artifactId>
    <version>1.0</version>
  </parent>
</project>`)

		if IsDept44Repo(dir) {
			t.Error("expected false for non-dept44 parent")
		}
	})
}

// ── TestPomVersion ──────────────────────────────────────────────────

func TestPomVersion(t *testing.T) {
	t.Run("normal version", func(t *testing.T) {
		dir := t.TempDir()
		writePom(t, dir, validPom)

		got := PomVersion(dir)
		if got != "1.2.3" {
			t.Errorf("got %q, want 1.2.3", got)
		}
	})

	t.Run("placeholder version", func(t *testing.T) {
		dir := t.TempDir()
		writePom(t, dir, `<project><version>@project.version@</version></project>`)

		got := PomVersion(dir)
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("empty version", func(t *testing.T) {
		dir := t.TempDir()
		writePom(t, dir, `<project><version></version></project>`)

		got := PomVersion(dir)
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}

// ── TestParentVersion ───────────────────────────────────────────────

func TestParentVersion(t *testing.T) {
	t.Run("with parent", func(t *testing.T) {
		dir := t.TempDir()
		writePom(t, dir, validPom)

		got := ParentVersion(dir)
		if got != "5.0.0" {
			t.Errorf("got %q, want 5.0.0", got)
		}
	})

	t.Run("without parent", func(t *testing.T) {
		dir := t.TempDir()
		writePom(t, dir, `<project><version>1.0</version></project>`)

		got := ParentVersion(dir)
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}

// ── TestInputSpecs ──────────────────────────────────────────────────

func TestInputSpecs(t *testing.T) {
	t.Run("with openapi-generator plugin", func(t *testing.T) {
		dir := t.TempDir()
		writePom(t, dir, `<?xml version="1.0" encoding="UTF-8"?>
<project>
  <build>
    <plugins>
      <plugin>
        <executions>
          <execution>
            <configuration>
              <inputSpec>${project.basedir}/src/main/resources/integrations/messaging-api-7.9.yaml</inputSpec>
            </configuration>
          </execution>
          <execution>
            <configuration>
              <inputSpec>${project.basedir}/src/main/resources/integrations/case-data-1.0.yml</inputSpec>
            </configuration>
          </execution>
        </executions>
      </plugin>
    </plugins>
  </build>
</project>`)

		specs := InputSpecs(dir)
		if len(specs) != 2 {
			t.Fatalf("got %d specs, want 2", len(specs))
		}
		if specs[0] != "messaging-api-7.9.yaml" {
			t.Errorf("specs[0] = %q", specs[0])
		}
		if specs[1] != "case-data-1.0.yml" {
			t.Errorf("specs[1] = %q", specs[1])
		}
	})

	t.Run("no build section", func(t *testing.T) {
		dir := t.TempDir()
		writePom(t, dir, `<project><version>1.0</version></project>`)

		specs := InputSpecs(dir)
		if len(specs) != 0 {
			t.Errorf("got %d specs, want 0", len(specs))
		}
	})
}

// ── TestExtractInputSpecs ───────────────────────────────────────────

func TestExtractInputSpecs(t *testing.T) {
	t.Run("XML plugin executions", func(t *testing.T) {
		pomData := []byte(`<project>
  <build>
    <plugins>
      <plugin>
        <executions>
          <execution>
            <configuration>
              <inputSpec>src/main/resources/integrations/foo-api-1.0.yaml</inputSpec>
            </configuration>
          </execution>
        </executions>
      </plugin>
    </plugins>
  </build>
</project>`)

		specs := extractInputSpecs(pomData)
		if len(specs) != 1 {
			t.Fatalf("got %d specs, want 1", len(specs))
		}
		if specs[0] != "foo-api-1.0.yaml" {
			t.Errorf("specs[0] = %q", specs[0])
		}
	})

	t.Run("regex fallback", func(t *testing.T) {
		// Non-standard XML that doesn't parse into our structs but has inputSpec tags
		pomData := []byte(`<project>
  <build>
    <pluginManagement>
      <plugins>
        <plugin>
          <configuration>
            <inputSpec>src/main/resources/integrations/bar-api-2.0.yml</inputSpec>
          </configuration>
        </plugin>
      </plugins>
    </pluginManagement>
  </build>
</project>`)

		specs := extractInputSpecs(pomData)
		if len(specs) != 1 {
			t.Fatalf("got %d specs, want 1", len(specs))
		}
		if specs[0] != "bar-api-2.0.yml" {
			t.Errorf("specs[0] = %q", specs[0])
		}
	})
}

// ── TestIsDept44Repo_MissingPom ─────────────────────────────────────

func TestIsDept44Repo_MissingPom(t *testing.T) {
	dir := t.TempDir()
	if IsDept44Repo(dir) {
		t.Error("expected false for missing pom.xml")
	}
}

// ── TestParentVersion_MissingPom ────────────────────────────────────

func TestParentVersion_MissingPom(t *testing.T) {
	dir := t.TempDir()
	got := ParentVersion(dir)
	if got != "" {
		t.Errorf("got %q, want empty for missing pom.xml", got)
	}
}

// ── TestInputSpecs_MissingPom ───────────────────────────────────────

func TestInputSpecs_MissingPom(t *testing.T) {
	dir := t.TempDir()
	specs := InputSpecs(dir)
	if len(specs) != 0 {
		t.Errorf("got %d specs, want 0 for missing pom.xml", len(specs))
	}
}
