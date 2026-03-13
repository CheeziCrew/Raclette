package tui

import (
	"strings"
	"testing"

	"github.com/CheeziCrew/raclette/internal/ops"
)

func TestDepVersion(t *testing.T) {
	tests := []struct {
		name string
		v    string
		want string
	}{
		{"empty", "", ""},
		{"property ref", "${some.version}", ""},
		{"literal", "1.0.0", "1.0.0"},
		{"snapshot", "2.0.0-SNAPSHOT", "2.0.0-SNAPSHOT"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := depVersion(tt.v)
			if got != tt.want {
				t.Errorf("depVersion(%q) = %q, want %q", tt.v, got, tt.want)
			}
		})
	}
}

func TestFormatDepByRepo(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := formatDepByRepo(nil)
		if result == "" {
			t.Error("expected non-empty string for no matches")
		}
		if !strings.Contains(result, "No matches") {
			t.Error("expected 'No matches' message")
		}
	})

	t.Run("with matches", func(t *testing.T) {
		matches := []ops.DepMatch{
			{Repo: "repo-b", GroupID: "com.b", Artifact: "art-b", Version: "2.0"},
			{Repo: "repo-a", GroupID: "com.a", Artifact: "art-a", Version: "1.0"},
		}
		result := formatDepByRepo(matches)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("with parent tag", func(t *testing.T) {
		matches := []ops.DepMatch{
			{Repo: "r", GroupID: "g", Artifact: "a", Version: "1.0", InParent: true},
		}
		result := formatDepByRepo(matches)
		if !strings.Contains(result, "[parent]") {
			t.Error("expected [parent] tag")
		}
	})

	t.Run("with managed version", func(t *testing.T) {
		matches := []ops.DepMatch{
			{Repo: "r", GroupID: "g", Artifact: "a", Version: ""},
		}
		result := formatDepByRepo(matches)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})
}

func TestFormatDepByDep(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := formatDepByDep(nil)
		if !strings.Contains(result, "No matches") {
			t.Error("expected 'No matches' message")
		}
	})

	t.Run("single group", func(t *testing.T) {
		matches := []ops.DepMatch{
			{Repo: "r1", GroupID: "g", Artifact: "a", Version: "1.0"},
			{Repo: "r2", GroupID: "g", Artifact: "a", Version: "2.0"},
		}
		result := formatDepByDep(matches)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("multiple groups", func(t *testing.T) {
		matches := []ops.DepMatch{
			{Repo: "r1", GroupID: "com.a", Artifact: "art-a", Version: "1.0"},
			{Repo: "r2", GroupID: "com.b", Artifact: "art-b", Version: "2.0"},
		}
		result := formatDepByDep(matches)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("with parent", func(t *testing.T) {
		matches := []ops.DepMatch{
			{Repo: "r", GroupID: "g", Artifact: "a", Version: "1.0", InParent: true},
		}
		result := formatDepByDep(matches)
		if !strings.Contains(result, "[parent]") {
			t.Error("expected [parent] tag")
		}
	})
}

func TestFormatStaleResults(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := formatStaleResults(nil)
		if !strings.Contains(result, "up to date") {
			t.Error("expected 'up to date' message")
		}
	})

	t.Run("with stale", func(t *testing.T) {
		stale := []ops.StaleSpec{
			{Repo: "r", SpecFile: "s.yaml", SpecVersion: "1.0", TargetRepo: "t", TargetVersion: "2.0"},
		}
		result := formatStaleResults(stale)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})
}

func TestFormatTransformResults(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := formatTransformResults(nil)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("with changed", func(t *testing.T) {
		results := []ops.Result{
			{Repo: "r1", Changed: true, Message: "updated"},
		}
		result := formatTransformResults(results)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("with error", func(t *testing.T) {
		results := []ops.Result{
			{Repo: "r1", Err: errStr("fail")},
		}
		result := formatTransformResults(results)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("unchanged with repo", func(t *testing.T) {
		results := []ops.Result{
			{Repo: "r1", Changed: false, Message: "no change"},
		}
		result := formatTransformResults(results)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("no repo message", func(t *testing.T) {
		results := []ops.Result{
			{Message: "All done."},
		}
		result := formatTransformResults(results)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})
}

func TestGroupMatchesByDep(t *testing.T) {
	matches := []ops.DepMatch{
		{Repo: "r1", GroupID: "g", Artifact: "a"},
		{Repo: "r2", GroupID: "g", Artifact: "a"},
		{Repo: "r3", GroupID: "g2", Artifact: "b"},
	}
	groups, order := groupMatchesByDep(matches)
	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}
	if len(order) != 2 {
		t.Errorf("expected 2 order entries, got %d", len(order))
	}
	// Order should be sorted
	if order[0] > order[1] {
		t.Errorf("expected sorted order, got %v", order)
	}
}

func TestDepHeader(t *testing.T) {
	h := depHeader(5, "by repo")
	if h == "" {
		t.Error("expected non-empty header")
	}
}

type errStr string

func (e errStr) Error() string { return string(e) }
