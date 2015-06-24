package roll

import (
	"testing"
)

func TestBuildStack(t *testing.T) {
	frame := buildStack(1)[0]
	if frame.Filename != "github.com/stvp/roll/stack_test.go" {
		t.Errorf("got: %s", frame.Filename)
	}
	if frame.Method != "roll.TestBuildStack" {
		t.Errorf("got: %s", frame.Method)
	}
	if frame.Line != 8 {
		t.Errorf("got: %d", frame.Line)
	}
}

func TestStackFingerprint(t *testing.T) {
	tests := []struct {
		Fingerprint string
		Title       string
		Stack       stack
	}{
		{
			"c9dfdc0e",
			"broken",
			stack{
				frame{"foo.go", "Oops", 1},
			},
		},
		{
			"21037bf5",
			"very broken",
			stack{
				frame{"foo.go", "Oops", 1},
			},
		},
		{
			"50d68db4",
			"broken",
			stack{
				frame{"foo.go", "Oops", 2},
			},
		},
		{
			"b341ee82",
			"broken",
			stack{
				frame{"foo.go", "Oops", 1},
				frame{"foo.go", "Oops", 2},
			},
		},
	}

	for i, test := range tests {
		fp := stackFingerprint(test.Title, test.Stack)
		if fp != test.Fingerprint {
			t.Errorf("tests[%d]: got %s", i, fp)
		}
	}
}

func TestShortenFilePath(t *testing.T) {
	tests := []struct {
		Given    string
		Expected string
	}{
		{"", ""},
		{"foo.go", "foo.go"},
		{"/usr/local/go/src/pkg/runtime/proc.c", "pkg/runtime/proc.c"},
		{"/home/foo/go/src/github.com/stvp/rollbar.go", "github.com/stvp/rollbar.go"},
	}
	for i, test := range tests {
		got := shortenFilePath(test.Given)
		if got != test.Expected {
			t.Errorf("tests[%d]: got %s", i, got)
		}
	}
}
