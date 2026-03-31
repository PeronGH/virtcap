package preset

import "testing"

func TestParseKnownPreset(t *testing.T) {
	definition, err := Parse("3:2-large")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if definition.Width != 2496 || definition.Height != 1664 || definition.Hz != 60 {
		t.Fatalf("Parse() dimensions = %dx%d@%d", definition.Width, definition.Height, definition.Hz)
	}
}

func TestParseAdditionalBuiltInPreset(t *testing.T) {
	definition, err := Parse("uwqhd")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if definition.Width != 3440 || definition.Height != 1440 || definition.Hz != 60 {
		t.Fatalf("Parse() dimensions = %dx%d@%d", definition.Width, definition.Height, definition.Hz)
	}
}

func TestParseUnknownPreset(t *testing.T) {
	if _, err := Parse("bogus"); err == nil {
		t.Fatalf("Parse() error = nil, want failure")
	}
}
