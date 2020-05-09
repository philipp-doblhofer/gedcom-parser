package gedcom

import (
	"testing"
)

var gedcomLines = []string{
	"0 HEAD",
	"0 @1@ INDI",
	"1 NAME Robert Eugene/Williams/",
	"1 SEX M",
	"1 BIRT",
	"2 DATE 02 OCT 1822",
	"1 FAMC @4@",
}

func TestLine_Level(t *testing.T) {
	expectedLevels := []uint8{0, 0, 1, 1, 1, 2, 1}
	for i, line := range gedcomLines {
		l := NewLine(&line)
		if *l.Level() != expectedLevels[i] {
			t.Errorf("unexpected level at index %d, expected: %d, actual: %d", i, expectedLevels[i], *l.Level())
		}
	}
}

func TestLine_XRefID(t *testing.T) {
	expectedValues := []string{"", "@1@", "", "", "", "", ""}
	for i, line := range gedcomLines {
		l := NewLine(&line)
		if l.XRefID() != nil && *l.XRefID() != expectedValues[i] {
			t.Errorf("unexpected XRefId at index %d, expected: %s, actual: %s", i, expectedValues[i], *l.XRefID())
		} else if l.XRefID() == nil && expectedValues[i] != "" {
			t.Errorf("unexpected XRefId at index %d, expected: %s, actual: nil", i, expectedValues[i])
		}
	}
}

func TestLine_Tag(t *testing.T) {
	expectedValues := []string{"HEAD", "INDI", "NAME", "SEX", "BIRT", "DATE", "FAMC"}
	for i, line := range gedcomLines {
		l := NewLine(&line)
		if *l.Tag() != expectedValues[i] {
			t.Errorf("unexpected tag at index %d, expected: %s, actual: %s", i, expectedValues[i], *l.Tag())
		}
	}
}

func TestLine_Value(t *testing.T) {
	expectedValues := []string{"", "", "Robert Eugene/Williams/", "M", "", "02 OCT 1822", "@4@"}
	for i, line := range gedcomLines {
		l := NewLine(&line)
		if l.Value() != nil && *l.Value() != expectedValues[i] {
			t.Errorf("unexpected value at index %d, expected: %s, actual: %s", i, expectedValues[i], *l.Value())
		} else if l.Value() == nil && expectedValues[i] != "" {
			t.Errorf("unexpected value at index %d, expected: %s, actual: nil", i, expectedValues[i])
		}
	}
}
