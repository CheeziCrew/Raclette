package ops

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// SpecPaths holds the generated and committed spec paths for a repo.
type SpecPaths struct {
	TargetFile   string // e.g. "target/openapi.yml"
	ResourceFile string // e.g. "src/integration-test/resources/openapi/openapi.yaml"
}

// ParseSpecPaths reads OpenApiSpecificationIT.java and extracts where
// the generated spec is written and where the committed spec lives.
func ParseSpecPaths(repoDir string) (SpecPaths, error) {
	testFile, err := findSpecTestFile(repoDir)
	if err != nil {
		return SpecPaths{}, err
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		return SpecPaths{}, fmt.Errorf("read test file: %w", err)
	}
	content := string(data)

	target := parseTargetPath(content)
	if target == "" {
		return SpecPaths{}, fmt.Errorf("could not find writeString target path in %s", filepath.Base(testFile))
	}

	resource := parseResourcePath(content, repoDir)
	if resource == "" {
		return SpecPaths{}, fmt.Errorf("could not find classpath resource in %s", filepath.Base(testFile))
	}

	return SpecPaths{
		TargetFile:   target,
		ResourceFile: resource,
	}, nil
}

// OwnSpecCurrent returns true if the repo's own OpenAPI spec version
// already matches its pom.xml version — meaning no update is needed.
func OwnSpecCurrent(repoDir string) bool {
	pomVer := PomVersion(repoDir)
	if pomVer == "" {
		return false
	}
	ownSpec := findOwnSpec(repoDir)
	if ownSpec == "" {
		return false
	}
	return extractSpecVersion(ownSpec) == pomVer
}

// CopySpec copies the generated spec over the committed one.
func CopySpec(repoDir string, paths SpecPaths) error {
	src := filepath.Join(repoDir, paths.TargetFile)
	dst := filepath.Join(repoDir, paths.ResourceFile)

	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read generated spec: %w", err)
	}

	if err := os.WriteFile(dst, data, 0644); err != nil {
		return fmt.Errorf("write spec: %w", err)
	}

	return nil
}

// findSpecTestFile locates OpenApiSpecificationIT.java in the repo.
func findSpecTestFile(repoDir string) (string, error) {
	var found string
	itRoot := filepath.Join(repoDir, "src", "integration-test", "java")

	err := filepath.Walk(itRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && info.Name() == "OpenApiSpecificationIT.java" {
			found = path
			return filepath.SkipAll
		}
		return nil
	})

	if err != nil || found == "" {
		return "", fmt.Errorf("no OpenApiSpecificationIT.java found")
	}
	return found, nil
}

// parseTargetPath extracts the Path.of("target/...") argument.
var targetPathRe = regexp.MustCompile(`writeString\s*\(\s*Path\.of\s*\(\s*"([^"]+)"`)

func parseTargetPath(content string) string {
	if m := targetPathRe.FindStringSubmatch(content); len(m) > 1 {
		return m[1]
	}
	return ""
}

// parseResourcePath extracts the classpath resource and resolves it to a real path.
// e.g. "classpath:/api/openapi.yaml" → "src/integration-test/resources/api/openapi.yaml"
var classpathRe = regexp.MustCompile(`@Value\s*\(\s*"classpath:/?([^"]+)"\s*\)`)

func parseResourcePath(content string, repoDir string) string {
	if m := classpathRe.FindStringSubmatch(content); len(m) > 1 {
		roots := []string{
			filepath.Join("src", "integration-test", "resources"),
			filepath.Join("src", "main", "resources"),
			filepath.Join("src", "test", "resources"),
		}
		for _, root := range roots {
			candidate := filepath.Join(root, m[1])
			if _, err := os.Stat(filepath.Join(repoDir, candidate)); err == nil {
				return candidate
			}
		}
	}
	return ""
}
