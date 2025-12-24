package posixflag

import (
	"errors"
	"testing"
)

func TestFlagSet_boolFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{
			name: "short flag",
			args: []string{"-v"},
			want: true,
		},
		{
			name: "long flag",
			args: []string{"--verbose"},
			want: true,
		},
		{
			name: "combined short flags",
			args: []string{"-vd"},
			want: true,
		},
		{
			name: "no flag",
			args: []string{},
			want: false,
		},
		{
			name: "long flag with explicit value",
			args: []string{"--verbose=true"},
			want: true,
		},
		{
			name: "long flag with false value",
			args: []string{"--verbose=false"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewFlagSet()

			var verbose bool
			fs.BoolVar(&verbose, "verbose", 'v', false, "verbose output")

			var debug bool
			fs.BoolVar(&debug, "debug", 'd', false, "debug mode")

			if err := fs.Parse(tt.args); err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if verbose != tt.want {
				t.Errorf("got verbose = %v, want %v", verbose, tt.want)
			}
		})
	}
}

func TestFlagSet_stringFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "short flag with value",
			args: []string{"-f", "file.txt"},
			want: "file.txt",
		},
		{
			name: "long flag with value",
			args: []string{"--file", "file.txt"},
			want: "file.txt",
		},
		{
			name: "long flag with equals",
			args: []string{"--file=file.txt"},
			want: "file.txt",
		},
		{
			name: "short flag with attached value",
			args: []string{"-ffile.txt"},
			want: "file.txt",
		},
		{
			name: "no flag",
			args: []string{},
			want: "default.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewFlagSet()
			var file string

			fs.StringVar(&file, "file", 'f', "default.txt", "input file")

			if err := fs.Parse(tt.args); err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if file != tt.want {
				t.Errorf("got file = %v, want %v", file, tt.want)
			}
		})
	}
}

func TestFlagSet_stringFlags_error(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "missing value",
			args: []string{"-f"},
		},
		{
			name: "missing value long flag",
			args: []string{"--file"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewFlagSet()
			var file string

			fs.StringVar(&file, "file", 'f', "default.txt", "input file")

			if err := fs.Parse(tt.args); err == nil {
				t.Error("got nil error, want non-nil")
			}
		})
	}
}

func TestFlagSet_combinedFlags(t *testing.T) {
	fs := NewFlagSet()
	var verbose, debug, all bool
	var file string

	fs.BoolVar(&verbose, "verbose", 'v', false, "verbose output")
	fs.BoolVar(&debug, "debug", 'd', false, "debug mode")
	fs.BoolVar(&all, "all", 'a', false, "show all")
	fs.StringVar(&file, "file", 'f', "", "input file")

	if err := fs.Parse([]string{"-vda", "-f", "test.txt"}); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if !verbose {
		t.Errorf("got verbose = false, want true")
	}
	if !debug {
		t.Errorf("got debug = false, want true")
	}
	if !all {
		t.Errorf("got all = false, want true")
	}
	if file != "test.txt" {
		t.Errorf("got file = %v, want test.txt", file)
	}
}

func TestFlagSet_args(t *testing.T) {
	fs := NewFlagSet()
	var verbose bool

	fs.BoolVar(&verbose, "verbose", 'v', false, "verbose output")

	if err := fs.Parse([]string{"-v", "arg1", "arg2", "--", "-not-a-flag"}); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	args := fs.Args()
	want := []string{"arg1", "arg2", "-not-a-flag"}

	if len(args) != len(want) {
		t.Fatalf("got %d args, want %d", len(args), len(want))
	}

	for i, arg := range args {
		if arg != want[i] {
			t.Errorf("args[%d]: got %v, want %v", i, arg, want[i])
		}
	}
}

func TestFlagSet_unknownFlag_error(t *testing.T) {
	fs := NewFlagSet()
	var verbose bool

	fs.BoolVar(&verbose, "verbose", 'v', false, "verbose output")

	err := fs.Parse([]string{"-x"})
	if err == nil {
		t.Fatal("got nil error for unknown flag, want non-nil")
	}

	if !errors.Is(err, ErrUnknownFlag) {
		t.Errorf("got %v, want ErrUnknownFlag", err)
	}
}

func TestFlagSet_lookup(t *testing.T) {
	fs := NewFlagSet()
	var verbose bool

	fs.BoolVar(&verbose, "verbose", 'v', false, "verbose output")

	flag := fs.Lookup("verbose")
	if flag == nil {
		t.Fatal("got nil flag for verbose, want non-nil")
	}

	if flag.Name != "verbose" {
		t.Errorf("got name = %v, want verbose", flag.Name)
	}

	if flag.Short != 'v' {
		t.Errorf("got short = %v, want v", flag.Short)
	}

	missingFlag := fs.Lookup("missing")
	if missingFlag != nil {
		t.Errorf("got %v for missing flag, want nil", missingFlag)
	}
}

func TestFlagSet_parsed(t *testing.T) {
	fs := NewFlagSet()
	var verbose bool

	fs.BoolVar(&verbose, "verbose", 'v', false, "verbose output")

	if fs.Parsed() {
		t.Error("Parsed() = true before Parse(), want false")
	}

	if err := fs.Parse([]string{"-v"}); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if !fs.Parsed() {
		t.Error("Parsed() = false after Parse(), want true")
	}
}

func TestFlagSet_emptyArgs(t *testing.T) {
	fs := NewFlagSet()
	var verbose bool

	fs.BoolVar(&verbose, "verbose", 'v', false, "verbose output")

	if fs.Args() != nil {
		t.Error("Args() != nil before Parse(), want nil")
	}

	if err := fs.Parse([]string{}); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	args := fs.Args()
	if args == nil {
		t.Fatal("Args() = nil after Parse(), want non-nil")
	}

	if len(args) != 0 {
		t.Errorf("got args = %v, want empty slice", args)
	}
}
