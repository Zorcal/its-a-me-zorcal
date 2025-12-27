package termui

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/zorcal/its-a-me-zorcal/core/termfs"
	"github.com/zorcal/its-a-me-zorcal/pkg/github"
)

func setupTest() (*termfs.FS, *mockSessionManager) {
	repos := []github.Repository{
		{
			Name:        "test-repo",
			Description: "A test repository",
			Language:    "Go",
			Stars:       42,
			URL:         "https://github.com/test/test-repo",
			UpdatedAt:   "2024-01-01T00:00:00Z",
		},
	}

	tfs := termfs.New(repos)

	tfs.AddFile("home/zorcal/projects/app.js", []byte("console.log('hello world');\n\n**URL:** https://github.com/example/app-js"))

	sessMgr := newMockSessionManager()

	return tfs, sessMgr
}

func TestChangeDirectory(t *testing.T) {
	tfs, sessMgr := setupTest()
	sessionID := "session1"

	tests := []struct {
		name       string
		startDir   string
		targetPath string
		wantDir    string
	}{
		{
			name:       "to home directory with tilde",
			startDir:   "home/guest", // default start
			targetPath: "~",
			wantDir:    "home/guest",
		},
		{
			name:       "to existing directory from root",
			startDir:   "",
			targetPath: "home",
			wantDir:    "home",
		},
		{
			name:       "to existing subdirectory",
			startDir:   "home",
			targetPath: "zorcal",
			wantDir:    "home/zorcal",
		},
		{
			name:       "to home with no arguments",
			startDir:   "home/zorcal",
			targetPath: "",
			wantDir:    "home/guest",
		},
		{
			name:       "simple parent from projects",
			startDir:   "home/zorcal/projects",
			targetPath: "..",
			wantDir:    "home/zorcal",
		},
		{
			name:       "double parent from projects",
			startDir:   "home/zorcal/projects",
			targetPath: "../..",
			wantDir:    "home",
		},
		{
			name:       "navigate to sibling directory via parent",
			startDir:   "home/zorcal",
			targetPath: "../guest",
			wantDir:    "home/guest",
		},
		{
			name:       "complex path with parent references",
			startDir:   "home/zorcal/projects",
			targetPath: "../../../home/zorcal",
			wantDir:    "home/zorcal",
		},
		{
			name:       "absolute path with parent references",
			startDir:   "home/guest",
			targetPath: "/home/zorcal/../guest",
			wantDir:    "home/guest",
		},
		{
			name:       "mixed current and parent directory references",
			startDir:   "home",
			targetPath: "./zorcal/../guest/./",
			wantDir:    "home/guest",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessMgr.SetCurrentDir(sessionID, tt.startDir)

			args := []string{}
			if tt.targetPath != "" {
				args = []string{tt.targetPath}
			}

			if _, err := ChangeDirectory(tfs, sessMgr, sessionID, args); err != nil {
				t.Fatalf("ChangeDirectory(tfs, sessMgr, %q, %v) error = %v, want nil", sessionID, args, err)
			}

			got := sessMgr.GetCurrentDir(sessionID)
			if want := tt.wantDir; got != want {
				t.Errorf("ChangeDirectory from %q to %q: current dir = %q, want %q", tt.startDir, tt.targetPath, got, want)
			}
		})
	}
}

func TestChangeDirectory_error(t *testing.T) {
	tfs, sessMgr := setupTest()
	sessionID := "session1"

	t.Run("nonexistent directory", func(t *testing.T) {
		in := "nonexistent"
		_, gotErr := ChangeDirectory(tfs, sessMgr, sessionID, []string{in})
		if gotErr == nil {
			t.Fatalf("ChangeDirectory(tfs, sessMgr, %q, %v) error = nil, want error", sessionID, []string{in})
		}
		if !errors.Is(gotErr, ErrFileNotFound) {
			t.Errorf("ChangeDirectory(tfs, sessMgr, %q, %v) error = %v, want %v", sessionID, []string{in}, gotErr, ErrFileNotFound)
		}
	})

	t.Run("target is file not directory", func(t *testing.T) {
		sessMgr.SetCurrentDir(sessionID, "")

		in := "home/zorcal/projects"
		if _, err := ChangeDirectory(tfs, sessMgr, sessionID, []string{in}); err != nil {
			t.Fatalf("setup: ChangeDirectory(tfs, sessMgr, %q, %v) error = %v, want nil", sessionID, []string{in}, err)
		}

		in = "test-repo.md"
		gotContext, gotErr := ChangeDirectory(tfs, sessMgr, sessionID, []string{in})
		if gotErr == nil {
			t.Fatalf("ChangeDirectory(tfs, sessMgr, %q, %v) error = nil, want error", sessionID, []string{in})
		}
		if !errors.Is(gotErr, ErrNotDirectory) {
			t.Errorf("ChangeDirectory(tfs, sessMgr, %q, %v) error = %v, want %v", sessionID, []string{in}, gotErr, ErrNotDirectory)
		}
		if wantContext := "test-repo.md"; gotContext != wantContext {
			t.Errorf("ChangeDirectory(tfs, sessMgr, %q, %v) context = %q, want %q", sessionID, []string{in}, gotContext, wantContext)
		}
	})
}

func TestListDirectoryContents(t *testing.T) {
	tfs, sessMgr := setupTest()
	sessionID := "session1"

	tests := []struct {
		name        string
		startDir    string
		args        []string
		wantSubstrs []string
	}{
		{
			name:        "root directory",
			startDir:    "",
			args:        []string{},
			wantSubstrs: []string{"home"},
		},
		{
			name:        "projects directory",
			startDir:    "home/zorcal/projects",
			args:        []string{},
			wantSubstrs: []string{"test-repo.md"},
		},
		{
			name:        "home directory",
			startDir:    "home",
			args:        []string{},
			wantSubstrs: []string{"guest", "zorcal"},
		},
		{
			name:        "path argument",
			startDir:    "home",
			args:        []string{"zorcal"},
			wantSubstrs: []string{"projects"},
		},
		{
			name:        "files as target",
			startDir:    "home/zorcal/projects",
			args:        []string{"test-repo.md"},
			wantSubstrs: []string{"test-repo.md"},
		},
		{
			name:        "combined flags",
			startDir:    "home/zorcal",
			args:        []string{"-la"},
			wantSubstrs: []string{"-c-  .secret.txt", "d--  projects/"},
		},
		{
			name:        "combined flags in reverse order",
			startDir:    "home/zorcal",
			args:        []string{"-al"},
			wantSubstrs: []string{"-c-  .secret.txt", "d--  projects/"},
		},
		{
			name:        "path before flags",
			startDir:    "home",
			args:        []string{"zorcal", "-l"},
			wantSubstrs: []string{"d--  projects/"},
		},
		{
			name:        "path between flags",
			startDir:    "home",
			args:        []string{"-a", "zorcal", "-l"},
			wantSubstrs: []string{"-c-  .secret.txt", "d--  projects/"},
		},
		{
			name:        "flags after path",
			startDir:    "home",
			args:        []string{"zorcal", "-al"},
			wantSubstrs: []string{"-c-  .secret.txt", "d--  projects/"},
		},
		{
			name:        "with path and flags",
			startDir:    "home",
			args:        []string{"-l", "zorcal"},
			wantSubstrs: []string{"d--  projects/"},
		},
		{
			name:        "list parent directory from projects",
			startDir:    "home/zorcal/projects",
			args:        []string{".."},
			wantSubstrs: []string{"projects"},
		},
		{
			name:        "list grandparent directory from projects",
			startDir:    "home/zorcal/projects",
			args:        []string{"../.."},
			wantSubstrs: []string{"zorcal"},
		},
		{
			name:        "complex path with parent references",
			startDir:    "home/guest",
			args:        []string{"../zorcal/../zorcal"},
			wantSubstrs: []string{"projects"},
		},
		{
			name:        "mixed current and parent directory references",
			startDir:    "home/zorcal",
			args:        []string{"./projects/../projects"},
			wantSubstrs: []string{"test-repo.md"},
		},
		{
			name:        "list sibling directory via parent",
			startDir:    "home/zorcal",
			args:        []string{"../guest"},
			wantSubstrs: []string{},
		},
		{
			name:        "absolute path with parent references",
			startDir:    "home/guest",
			args:        []string{"/home/zorcal/../guest"},
			wantSubstrs: []string{},
		},
		{
			name:        "filetype indicators in projects directory",
			startDir:    "home/zorcal/projects",
			args:        []string{"-l"},
			wantSubstrs: []string{"-co  test-repo.md", "-co  app.js"},
		},
		{
			name:        "filetype indicators outside projects directory",
			startDir:    "home/zorcal",
			args:        []string{"-al"},
			wantSubstrs: []string{"-c-  .secret.txt", "d--  projects/"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessMgr.SetCurrentDir(sessionID, tt.startDir)

			got, err := ListDirectoryContents(tfs, sessMgr, sessionID, tt.args)
			if err != nil {
				t.Fatalf("ListDirectoryContents(tfs, sessMgr, %q, %v) error = %v, want nil", sessionID, tt.args, err)
			}

			var missingSubstrs []string
			for _, s := range tt.wantSubstrs {
				if !strings.Contains(got, s) {
					missingSubstrs = append(missingSubstrs, strconv.Quote(s))
				}
			}
			if len(missingSubstrs) > 0 {
				t.Errorf("ListDirectoryContents(tfs, sessMgr, %q, %v) output = %q, want to contain sub strings %q", sessionID, tt.args, got, tt.wantSubstrs)
			}
		})
	}

	t.Run("hidden files", func(t *testing.T) {
		tests := []struct {
			name       string
			args       []string
			wantHidden bool
		}{
			{
				name:       "hides hidden files by default",
				args:       []string{},
				wantHidden: false,
			},
			{
				name:       "with -a flag shows hidden files",
				args:       []string{"-a"},
				wantHidden: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				sessMgr.SetCurrentDir(sessionID, "home/zorcal")

				got, err := ListDirectoryContents(tfs, sessMgr, sessionID, tt.args)
				if err != nil {
					t.Fatalf("ListDirectoryContents(tfs, sessMgr, %q, %v) error = %v, want nil", sessionID, tt.args, err)
				}

				hiddenFilePresent := strings.Contains(got, ".secret.txt")
				if tt.wantHidden != hiddenFilePresent {
					t.Errorf("ListDirectoryContents(tfs, sessMgr, %q, %v) output = %q, want to contain hidden file .secret.txt is %v", sessionID, tt.args, got, tt.wantHidden)
				}

				if !strings.Contains(got, "projects") {
					t.Errorf("ListDirectoryContents(tfs, sessMgr, %q, %v) output = %q, want to contain visible file projects", sessionID, tt.args, got)
				}
			})
		}
	})

	t.Run("newline formatting", func(t *testing.T) {
		sessMgr.SetCurrentDir(sessionID, "home/zorcal/projects")

		got, err := ListDirectoryContents(tfs, sessMgr, sessionID, []string{"-l"})
		if err != nil {
			t.Fatalf("ListDirectoryContents(tfs, sessMgr, %q, %v) error = %v, want nil", sessionID, []string{"-l"}, err)
		}

		// Should contain newlines between entries
		if !strings.Contains(got, "\n") {
			t.Errorf("ListDirectoryContents(tfs, sessMgr, %q, %v) output = %q, should contain newlines between entries", sessionID, []string{"-l"}, got)
		}

		// Should not have entries separated by spaces
		if strings.Contains(got, "app.js  -c-") || strings.Contains(got, "test-repo.md  -co") {
			t.Errorf("ListDirectoryContents(tfs, sessMgr, %q, %v) output = %q, should separate entries with newlines, not spaces", sessionID, []string{"-l"}, got)
		}
	})
}

func TestListDirectory_error(t *testing.T) {
	tfs, sessMgr := setupTest()
	sessionID := "session1"

	t.Run("nonexistent directory", func(t *testing.T) {
		gotContext, gotErr := ListDirectoryContents(tfs, sessMgr, sessionID, []string{"nonexistent"})
		if gotErr == nil {
			t.Fatalf("ListDirectory(tfs, sessMgr, %q, %v) error = nil, want error", sessionID, []string{"nonexistent"})
		}
		if !errors.Is(gotErr, ErrFileNotFound) {
			t.Errorf("ListDirectory(tfs, sessMgr, %q, %v) error = %v, want %v", sessionID, []string{"nonexistent"}, gotErr, ErrFileNotFound)
		}
		if wantContext := "nonexistent"; gotContext != wantContext {
			t.Errorf("ListDirectory(tfs, sessMgr, %q, %v) context = %q, want %q", sessionID, []string{"nonexistent"}, gotContext, wantContext)
		}
	})

	t.Run("multiple paths", func(t *testing.T) {
		_, gotErr := ListDirectoryContents(tfs, sessMgr, sessionID, []string{"home", "zorcal"})
		if gotErr == nil {
			t.Fatalf("ListDirectory(tfs, sessMgr, %q, %v) error = nil, want error", sessionID, []string{"home", "zorcal"})
		}
		if !errors.Is(gotErr, ErrTooManyArguments) {
			t.Errorf("ListDirectory(tfs, sessMgr, %q, %v) error = %v, want %v", sessionID, []string{"home", "zorcal"}, gotErr, ErrTooManyArguments)
		}
	})

	t.Run("multiple paths with flags", func(t *testing.T) {
		_, gotErr := ListDirectoryContents(tfs, sessMgr, sessionID, []string{"-l", "home", "zorcal"})
		if gotErr == nil {
			t.Fatalf("ListDirectory(tfs, sessMgr, %q, %v) error = nil, want error", sessionID, []string{"-l", "home", "zorcal"})
		}
		if !errors.Is(gotErr, ErrTooManyArguments) {
			t.Errorf("ListDirectory(tfs, sessMgr, %q, %v) error = %v, want %v", sessionID, []string{"-l", "home", "zorcal"}, gotErr, ErrTooManyArguments)
		}
	})

	t.Run("unknown flag error", func(t *testing.T) {
		_, gotErr := ListDirectoryContents(tfs, sessMgr, sessionID, []string{"-x"})
		if gotErr == nil {
			t.Fatalf("ListDirectory(tfs, sessMgr, %q, %v) error = nil, want error", sessionID, []string{"-x"})
		}
		if !errors.Is(gotErr, ErrInvalidFlag) {
			t.Errorf("ListDirectory(tfs, sessMgr, %q, %v) error = %v, want ErrInvalidFlag", sessionID, []string{"-x"}, gotErr)
		}
	})

	t.Run("invalid path arguments", func(t *testing.T) {
		tests := []struct {
			name string
			args []string
		}{
			{
				name: "path with newline",
				args: []string{"path\nwith\nnewline"},
			},
			{
				name: "path with carriage return",
				args: []string{"path\rwith\rcarriage"},
			},
			{
				name: "path with null byte",
				args: []string{"path\x00with\x00null"},
			},
			{
				name: "very long path",
				args: []string{strings.Repeat("a", 300)},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, gotErr := ListDirectoryContents(tfs, sessMgr, sessionID, tt.args)
				if gotErr == nil {
					t.Fatalf("ListDirectory(tfs, sessMgr, %q, %v) error = nil, want error", sessionID, tt.args)
				}
				if !errors.Is(gotErr, ErrInvalidFlag) {
					t.Errorf("ListDirectory(tfs, sessMgr, %q, %v) error = %v, want ErrInvalidFlag", sessionID, tt.args, gotErr)
				}
			})
		}
	})
}

func TestPrintWorkingDirectory(t *testing.T) {
	_, sessMgr := setupTest()
	sessionID := "session1"

	t.Run("at root", func(t *testing.T) {
		sessMgr.SetCurrentDir(sessionID, "")

		got, err := PrintWorkingDirectory(sessMgr, sessionID)
		if err != nil {
			t.Fatalf("PrintWorkingDirectory(sessMgr, %q) error = %v, want nil", sessionID, err)
		}
		if want := "/"; got != want {
			t.Errorf("PrintWorkingDirectory(sessMgr, %q) path = %q, want %q", sessionID, got, want)
		}
	})

	t.Run("in subdirectory", func(t *testing.T) {
		sessMgr.SetCurrentDir(sessionID, "home/zorcal")

		got, err := PrintWorkingDirectory(sessMgr, sessionID)
		if err != nil {
			t.Fatalf("PrintWorkingDirectory(sessMgr, %q) error = %v, want nil", sessionID, err)
		}
		if want := "/home/zorcal"; got != want {
			t.Errorf("PrintWorkingDirectory(sessMgr, %q) path = %q, want %q", sessionID, got, want)
		}
	})
}

func TestCatFile(t *testing.T) {
	tfs, sessMgr := setupTest()
	sessionID := "session1"

	t.Run("existing file", func(t *testing.T) {
		sessMgr.SetCurrentDir(sessionID, "home/zorcal/projects")

		got, err := CatFile(tfs, sessMgr, sessionID, []string{"test-repo.md"})
		if err != nil {
			t.Fatalf("CatFile(tfs, sessMgr, %q, %v) error = %v, want nil", sessionID, []string{"test-repo.md"}, err)
		}
		if want := "test-repo"; !strings.Contains(got, want) {
			t.Errorf("CatFile(tfs, sessMgr, %q, %v) content = %q, want to contain %q", sessionID, []string{"test-repo.md"}, got, want)
		}
		if want := "A test repository"; !strings.Contains(got, want) {
			t.Errorf("CatFile(tfs, sessMgr, %q, %v) content = %q, want to contain %q", sessionID, []string{"test-repo.md"}, got, want)
		}
	})
}

func TestCatFile_error(t *testing.T) {
	tfs, sessMgr := setupTest()
	sessionID := "session1"

	t.Run("nonexistent file", func(t *testing.T) {
		gotContext, gotErr := CatFile(tfs, sessMgr, sessionID, []string{"nonexistent.txt"})
		if gotErr == nil {
			t.Fatalf("CatFile(tfs, sessMgr, %q, %v) error = nil, want error", sessionID, []string{"nonexistent.txt"})
		}
		if !errors.Is(gotErr, ErrFileNotFound) {
			t.Errorf("CatFile(tfs, sessMgr, %q, %v) error = %v, want %v", sessionID, []string{"nonexistent.txt"}, gotErr, ErrFileNotFound)
		}
		if wantContext := "nonexistent.txt"; gotContext != wantContext {
			t.Errorf("CatFile(tfs, sessMgr, %q, %v) context = %q, want %q", sessionID, []string{"nonexistent.txt"}, gotContext, wantContext)
		}
	})

	t.Run("target is directory", func(t *testing.T) {
		sessMgr.SetCurrentDir(sessionID, "")

		gotContext, gotErr := CatFile(tfs, sessMgr, sessionID, []string{"home"})
		if gotErr == nil {
			t.Fatalf("CatFile(tfs, sessMgr, %q, %v) error = nil, want error", sessionID, []string{"home"})
		}
		if !errors.Is(gotErr, ErrIsDirectory) {
			t.Errorf("CatFile(tfs, sessMgr, %q, %v) error = %v, want %v", sessionID, []string{"home"}, gotErr, ErrIsDirectory)
		}
		if wantContext := "home"; gotContext != wantContext {
			t.Errorf("CatFile(tfs, sessMgr, %q, %v) context = %q, want %q", sessionID, []string{"home"}, gotContext, wantContext)
		}
	})

	t.Run("no argument provided", func(t *testing.T) {
		_, gotErr := CatFile(tfs, sessMgr, sessionID, []string{})
		if gotErr == nil {
			t.Fatalf("CatFile(tfs, sessMgr, %q, %v) error = nil, want error", sessionID, []string{})
		}
		if !errors.Is(gotErr, ErrMissingArgument) {
			t.Errorf("CatFile(tfs, sessMgr, %q, %v) error = %v, want %v", sessionID, []string{}, gotErr, ErrMissingArgument)
		}
	})
}

func TestResolvePath(t *testing.T) {
	t.Run("home directory shortcuts", func(t *testing.T) {
		tests := []struct {
			currentDir string
			targetPath string
			want       string
		}{
			{"home/zorcal", "~", "home/guest"},
			{"home/zorcal", "$HOME", "home/guest"},
			{"home/zorcal", ".", "home/zorcal"},
			{"home/zorcal", "..", "home"},
			{"", "..", ""},
			{".", "..", ""},
			{"home/zorcal/projects", "../..", "home"},
		}
		for _, tt := range tests {
			got := resolvePath(tt.currentDir, tt.targetPath)
			if got != tt.want {
				t.Errorf("resolvePath(%q, %q) result = %q, want %q", tt.currentDir, tt.targetPath, got, tt.want)
			}
		}
	})

	t.Run("absolute paths", func(t *testing.T) {
		tests := []struct {
			currentDir string
			targetPath string
			want       string
		}{
			{"home/zorcal", "/", ""},
			{"home/zorcal", "/home", "home"},
			{"home/zorcal", "/home/guest", "home/guest"},
		}
		for _, tt := range tests {
			got := resolvePath(tt.currentDir, tt.targetPath)
			if got != tt.want {
				t.Errorf("resolvePath(%q, %q) result = %q, want %q", tt.currentDir, tt.targetPath, got, tt.want)
			}
		}
	})

	t.Run("relative paths", func(t *testing.T) {
		tests := []struct {
			currentDir string
			targetPath string
			want       string
		}{
			{"home", "zorcal", "home/zorcal"},
			{"", "home", "home"},
			{".", "home", "home"},
			{"home/zorcal", "projects", "home/zorcal/projects"},
		}
		for _, tt := range tests {
			got := resolvePath(tt.currentDir, tt.targetPath)
			if got != tt.want {
				t.Errorf("resolvePath(%q, %q) result = %q, want %q", tt.currentDir, tt.targetPath, got, tt.want)
			}
		}
	})

	t.Run("paths with parent directory references", func(t *testing.T) {
		tests := []struct {
			currentDir string
			targetPath string
			want       string
		}{
			{"", "/home/zorcal/../zorcal", "home/zorcal"},
			{"home", "zorcal/../guest", "home/guest"},
			{"home/zorcal/projects", "../../guest", "home/guest"},
			{"home/zorcal", "../guest/documents", "home/guest/documents"},
			{"", "/home/../home/zorcal", "home/zorcal"},
			{"", "/home/../../home", "home"},
			{"home", "../../../home", "home"},
			{"home", "./zorcal/../guest/./documents", "home/guest/documents"},
			{"", "/home/zorcal/projects/../..", "home"},
			{"home/zorcal/projects", "..", "home/zorcal"},
			{"", "/home/zorcal/..", "home"},
		}
		for _, tt := range tests {
			got := resolvePath(tt.currentDir, tt.targetPath)
			if got != tt.want {
				t.Errorf("resolvePath(%q, %q) result = %q, want %q", tt.currentDir, tt.targetPath, got, tt.want)
			}
		}
	})
}

func TestGeneratePrompt(t *testing.T) {
	tests := []struct {
		currDir string
		want    string
	}{
		{"home/guest", "guest@machine:~$ "},
		{"", "guest@machine:/$ "},
		{"home", "guest@machine:/home$ "},
		{"home/zorcal/projects", "guest@machine:/home/zorcal/projects$ "},
		{"usr", "guest@machine:/usr$ "},
		{"var/log/app/debug", "guest@machine:/var/log/app/debug$ "},
	}
	for _, tt := range tests {
		got := GeneratePrompt(tt.currDir)
		if got != tt.want {
			t.Errorf("GeneratePrompt(%q) result = %q, want %q", tt.currDir, got, tt.want)
		}
	}
}

func TestOpenFile(t *testing.T) {
	tfs, sessMgr := setupTest()
	sessionID := "session1"

	tests := []struct {
		name     string
		startDir string
		args     []string
		wantURL  string
	}{
		{
			name:     "existing openable file",
			startDir: "home/zorcal/projects",
			args:     []string{"test-repo.md"},
			wantURL:  "https://github.com/test/test-repo",
		},
		{
			name:     "existing openable file with relative path",
			startDir: "home/zorcal",
			args:     []string{"projects/test-repo.md"},
			wantURL:  "https://github.com/test/test-repo",
		},
		{
			name:     "existing openable file with absolute path",
			startDir: "",
			args:     []string{"/home/zorcal/projects/app.js"},
			wantURL:  "https://github.com/example/app-js",
		},
		{
			name:     "existing openable file with parent directory navigation",
			startDir: "home/guest",
			args:     []string{"../zorcal/projects/test-repo.md"},
			wantURL:  "https://github.com/test/test-repo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessMgr.SetCurrentDir(sessionID, tt.startDir)

			got, err := OpenFile(tfs, sessMgr, sessionID, tt.args)
			if err != nil {
				t.Fatalf("OpenFile(tfs, sessMgr, %q, %v) error = %v, want nil", sessionID, tt.args, err)
			}
			if got != tt.wantURL {
				t.Errorf("OpenFile(tfs, sessMgr, %q, %v) url = %q, want %q", sessionID, tt.args, got, tt.wantURL)
			}
		})
	}
}

func TestOpenFile_error(t *testing.T) {
	tfs, sessMgr := setupTest()
	sessionID := "session1"

	tests := []struct {
		name        string
		startDir    string
		args        []string
		wantErr     error
		wantContext string
	}{
		{
			name:        "nonexistent file",
			startDir:    "",
			args:        []string{"nonexistent.md"},
			wantErr:     ErrNotOpenable,
			wantContext: "nonexistent.md",
		},
		{
			name:        "file without URL",
			startDir:    "home/zorcal",
			args:        []string{".secret.txt"},
			wantErr:     ErrNotOpenable,
			wantContext: ".secret.txt",
		},
		{
			name:        "directory instead of file",
			startDir:    "",
			args:        []string{"home"},
			wantErr:     ErrNotOpenable,
			wantContext: "home",
		},
		{
			name:        "no argument provided",
			startDir:    "",
			args:        []string{},
			wantErr:     ErrMissingArgument,
			wantContext: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessMgr.SetCurrentDir(sessionID, tt.startDir)

			gotContext, gotErr := OpenFile(tfs, sessMgr, sessionID, tt.args)
			if gotErr == nil {
				t.Fatalf("OpenFile(tfs, sessMgr, %q, %v) error = nil, want error", sessionID, tt.args)
			}
			if gotErr != tt.wantErr {
				t.Errorf("OpenFile(tfs, sessMgr, %q, %v) error = %v, want %v", sessionID, tt.args, gotErr, tt.wantErr)
			}
			if gotContext != tt.wantContext {
				t.Errorf("OpenFile(tfs, sessMgr, %q, %v) context = %q, want %q", sessionID, tt.args, gotContext, tt.wantContext)
			}
		})
	}
}

func TestIsValidPathArgument(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want bool
	}{
		{
			name: "valid simple path",
			arg:  "home",
			want: true,
		},
		{
			name: "valid relative path",
			arg:  "../home/user",
			want: true,
		},
		{
			name: "valid absolute path",
			arg:  "/home/user",
			want: true,
		},
		{
			name: "valid path with spaces",
			arg:  "my documents/file.txt",
			want: true,
		},
		{
			name: "valid current directory",
			arg:  ".",
			want: true,
		},
		{
			name: "valid parent directory",
			arg:  "..",
			want: true,
		},
		{
			name: "empty string",
			arg:  "",
			want: false,
		},
		{
			name: "path with newline",
			arg:  "path\nwith\nnewline",
			want: false,
		},
		{
			name: "path with carriage return",
			arg:  "path\rwith\rcarriage",
			want: false,
		},
		{
			name: "path with null byte",
			arg:  "path\x00with\x00null",
			want: false,
		},
		{
			name: "very long path",
			arg:  strings.Repeat("a", 300),
			want: false,
		},
		{
			name: "long but acceptable path",
			arg:  strings.Repeat("a", 100),
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidPathArgument(tt.arg)
			if got != tt.want {
				t.Errorf("isValidPathArgument(%q) = %v, want %v", tt.arg, got, tt.want)
			}
		})
	}
}
