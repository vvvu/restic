package backend

import (
	"testing"

	rtest "github.com/restic/restic/internal/test"
)

func TestShellSplitterEscaping(t *testing.T) {
	var tests = []struct {
		name string
		data string
		args []string
		err  string
	}{
		{
			name: "escaped space",
			data: `foo\ bar`,
			args: []string{"foo bar"},
		},
		{
			name: "backslash at end is error",
			data: `foo\`,
			err:  "backslash escape at end of string",
		},
		{
			name: "double backslash",
			data: `foo\\bar`,
			args: []string{"foo\\bar"},
		},
		{
			name: "escaped quote in word",
			data: `foo\"bar`,
			args: []string{"foo\"bar"},
		},
		{
			name: "multiple escaped spaces",
			data: `foo\ \ bar\ \ baz`,
			args: []string{"foo  bar  baz"},
		},
		{
			name: "backslash in double quotes (literal)",
			data: `"foo\ bar"`,
			args: []string{"foo bar"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args, err := SplitShellStrings(test.data)
			if test.err != "" {
				rtest.Assert(t, err != nil, "expected error: %s", test.err)
				rtest.Assert(t, err.Error() == test.err, "wrong error message: %v", err)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(args) != len(test.args) {
				t.Fatalf("wrong number of args, want %d, got %d: %#v", len(test.args), len(args), args)
			}
			for i, want := range test.args {
				if args[i] != want {
					t.Errorf("arg[%d] wrong, want %q, got %q", i, want, args[i])
				}
			}
		})
	}
}
