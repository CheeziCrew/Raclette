package maven

import "testing"

func TestCommands_NonEmpty(t *testing.T) {
	cmds := Commands()
	if len(cmds) == 0 {
		t.Fatal("Commands() returned empty slice")
	}
}

func TestCommands_AllHaveRequiredFields(t *testing.T) {
	for _, cmd := range Commands() {
		t.Run(cmd.Name, func(t *testing.T) {
			if cmd.Name == "" {
				t.Error("command has empty Name")
			}
			if cmd.Description == "" {
				t.Error("command has empty Description")
			}
			if cmd.Icon == "" {
				t.Error("command has empty Icon")
			}
		})
	}
}

func TestCommands_KindsAreValid(t *testing.T) {
	validKinds := map[Kind]bool{
		KindMaven:      true,
		KindScan:       true,
		KindTransform:  true,
		KindUpdateSpec: true,
	}

	for _, cmd := range Commands() {
		t.Run(cmd.Name, func(t *testing.T) {
			if !validKinds[cmd.Kind] {
				t.Errorf("command %q has invalid Kind %d", cmd.Name, cmd.Kind)
			}
		})
	}
}

func TestCommands_MavenKindHasArgs(t *testing.T) {
	for _, cmd := range Commands() {
		if cmd.Kind == KindMaven && len(cmd.Args) == 0 {
			t.Errorf("KindMaven command %q has no Args", cmd.Name)
		}
	}
}

func TestCommands_ScanKindExists(t *testing.T) {
	found := false
	for _, cmd := range Commands() {
		if cmd.Kind == KindScan {
			found = true
			break
		}
	}
	if !found {
		t.Error("no KindScan commands found")
	}
}

func TestCommands_TransformKindExists(t *testing.T) {
	found := false
	for _, cmd := range Commands() {
		if cmd.Kind == KindTransform {
			found = true
			break
		}
	}
	if !found {
		t.Error("no KindTransform commands found")
	}
}

func TestCommands_PromptsHaveKeys(t *testing.T) {
	for _, cmd := range Commands() {
		for _, p := range cmd.Prompts {
			if p.Key == "" {
				t.Errorf("command %q has prompt with empty Key", cmd.Name)
			}
			if p.Label == "" {
				t.Errorf("command %q has prompt with empty Label", cmd.Name)
			}
		}
	}
}

func TestKindConstants(t *testing.T) {
	// Verify iota ordering
	if KindMaven != 0 {
		t.Errorf("KindMaven = %d, want 0", KindMaven)
	}
	if KindScan != 1 {
		t.Errorf("KindScan = %d, want 1", KindScan)
	}
	if KindTransform != 2 {
		t.Errorf("KindTransform = %d, want 2", KindTransform)
	}
	if KindUpdateSpec != 3 {
		t.Errorf("KindUpdateSpec = %d, want 3", KindUpdateSpec)
	}
}
