package ops

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ── XML structs for pom.xml parsing ─────────────────────────────────

type pomProject struct {
	XMLName    xml.Name   `xml:"project"`
	Parent     pomParent  `xml:"parent"`
	GroupID    string     `xml:"groupId"`
	ArtifactID string    `xml:"artifactId"`
	Version    string     `xml:"version"`
	Build      *pomBuild  `xml:"build"`
}

type pomParent struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
}

type pomBuild struct {
	Plugins pomPlugins `xml:"plugins"`
}

type pomPlugins struct {
	Plugins []pomPlugin `xml:"plugin"`
}

type pomPlugin struct {
	Executions pomExecutions `xml:"executions"`
}

type pomExecutions struct {
	Executions []pomExecution `xml:"execution"`
}

type pomExecution struct {
	Configuration pomConfiguration `xml:"configuration"`
}

type pomConfiguration struct {
	InputSpec string `xml:"inputSpec"`
}

// ReadPom reads and parses a pom.xml from the given repo directory.
func ReadPom(repoPath string) (*pomProject, []byte, error) {
	data, err := os.ReadFile(filepath.Join(repoPath, "pom.xml"))
	if err != nil {
		return nil, nil, err
	}
	var pom pomProject
	if err := xml.Unmarshal(data, &pom); err != nil {
		return nil, data, err
	}
	return &pom, data, nil
}

// IsDept44Repo returns true if the pom.xml has a dept44 parent.
func IsDept44Repo(repoPath string) bool {
	pom, _, err := ReadPom(repoPath)
	if err != nil {
		return false
	}
	return strings.Contains(pom.Parent.ArtifactID, "dept44")
}

// PomVersion returns the project version from pom.xml.
func PomVersion(repoPath string) string {
	pom, _, err := ReadPom(repoPath)
	if err != nil {
		return ""
	}
	v := strings.TrimSpace(pom.Version)
	if v == "" || v == "@project.version@" {
		return ""
	}
	return v
}

// ParentVersion returns the parent version from pom.xml.
func ParentVersion(repoPath string) string {
	pom, _, err := ReadPom(repoPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(pom.Parent.Version)
}

// InputSpecs extracts inputSpec filenames from pom.xml openapi-generator executions.
func InputSpecs(repoPath string) []string {
	_, data, err := ReadPom(repoPath)
	if err != nil || data == nil {
		return nil
	}
	return extractInputSpecs(data)
}

func extractInputSpecs(pomData []byte) []string {
	var pom pomProject
	if err := xml.Unmarshal(pomData, &pom); err == nil && pom.Build != nil {
		var specs []string
		for _, plugin := range pom.Build.Plugins.Plugins {
			for _, exec := range plugin.Executions.Executions {
				if spec := exec.Configuration.InputSpec; spec != "" {
					specs = append(specs, filepath.Base(spec))
				}
			}
		}
		if len(specs) > 0 {
			return specs
		}
	}

	// Fallback: regex for poms with non-standard structure
	re := regexp.MustCompile(`<inputSpec>[^<]*?([^/<]+\.ya?ml)</inputSpec>`)
	matches := re.FindAllStringSubmatch(string(pomData), -1)
	var specs []string
	for _, m := range matches {
		specs = append(specs, m[1])
	}
	return specs
}
