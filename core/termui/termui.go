package termui

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/zorcal/its-a-me-zorcal/core/termfs"
	"github.com/zorcal/its-a-me-zorcal/pkg/posixflag"
)

// Terminal command errors.
var (
	ErrFileNotFound     = errors.New("file not found")
	ErrNotDirectory     = errors.New("not a directory")
	ErrIsDirectory      = errors.New("is a directory")
	ErrMissingArgument  = errors.New("missing argument")
	ErrTooManyArguments = errors.New("too many arguments")
	ErrAccessDenied     = errors.New("access denied")
	ErrInvalidFlag      = errors.New("invalid flag")
	ErrNotOpenable      = errors.New("not openable")
)

// SessionManager defines the interface for managing terminal sessions.
type SessionManager interface {
	GetCurrentDir(sessionID string) string
	SetCurrentDir(sessionID string, dir string)
}

// ChangeDirectory changes the current working directory for a session.
// Returns the target path and error. On success, returns ("", nil).
// On error, returns (targetPath, error) where targetPath is the path the user
// attempted to access, allowing the caller to format contextual error messages.
// Possible errors: ErrFileNotFound, ErrNotDirectory, ErrAccessDenied.
func ChangeDirectory(tfs *termfs.FS, sessMgr SessionManager, sessionID string, args []string) (string, error) {
	var targetPath string
	if len(args) == 0 {
		// No arguments goes to home
		targetPath = "~"
	} else {
		targetPath = args[0]
	}

	currDir := sessMgr.GetCurrentDir(sessionID)
	newDir := resolvePath(currDir, targetPath)

	openPath := newDir
	if openPath == "" {
		openPath = "."
	}

	info, err := fs.Stat(tfs, openPath)
	if err != nil {
		return targetPath, fmt.Errorf("stat directory %q: %w", openPath, mapFSErr(err))
	}

	if !info.IsDir() {
		return targetPath, ErrNotDirectory
	}

	sessMgr.SetCurrentDir(sessionID, newDir)

	return "", nil
}

// ListDirectoryContents lists the contents of a directory.
// Returns directory listing and error. On success, returns (output, nil) where output
// contains the formatted directory listing. On error, returns (contextInfo, error)
// where contextInfo is the path/argument that caused the error.
// Possible errors: ErrFileNotFound, ErrTooManyArguments, ErrAccessDenied.
func ListDirectoryContents(tfs *termfs.FS, sessMgr SessionManager, sessionID string, args []string) (string, error) {
	currDir := sessMgr.GetCurrentDir(sessionID)

	// Parse flags and path
	flagSet := posixflag.NewFlagSet()
	var showAll, longList bool
	flagSet.BoolVar(&showAll, "all", 'a', false, "show hidden files")
	flagSet.BoolVar(&longList, "long", 'l', false, "long listing format")

	if err := flagSet.Parse(args); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidFlag, err)
	}

	remaining := flagSet.Args()
	if len(remaining) > 1 {
		return "", ErrTooManyArguments
	}

	targetPath := currDir
	if len(remaining) == 1 {
		// For ls command, the single remaining argument should be a path
		// Validate it looks like a reasonable path before using it
		pathArg := remaining[0]
		if !isValidPathArgument(pathArg) {
			return "", fmt.Errorf("%w: invalid path argument", ErrInvalidFlag)
		}
		targetPath = resolvePath(currDir, pathArg)
	}

	openPath := targetPath
	if openPath == "" {
		openPath = "."
	}

	info, err := fs.Stat(tfs, openPath)
	if err != nil {
		// We know exactly what path argument was provided since we validated it
		remaining := flagSet.Args()
		wrappedErr := fmt.Errorf("stat path %q: %w", openPath, mapFSErr(err))
		if len(remaining) == 1 {
			return remaining[0], wrappedErr
		}
		// If no path argument, return the current directory for context
		return ".", wrappedErr
	}

	if !info.IsDir() {
		return info.Name(), nil
	}

	entries, err := fs.ReadDir(tfs, openPath)
	if err != nil {
		return "", fmt.Errorf("read directory %q: %w", openPath, mapFSErr(err))
	}

	// Filter hidden files unless -a is used
	if !showAll {
		entries = slices.DeleteFunc(entries, func(entry fs.DirEntry) bool {
			return strings.HasPrefix(entry.Name(), ".")
		})
	}

	if len(entries) == 0 {
		return "", nil
	}

	var output strings.Builder
	for i, entry := range entries {
		if i > 0 {
			if longList {
				output.WriteString("\n")
			} else {
				output.WriteString("  ")
			}
		}

		name := entry.Name()

		// Long format: show 3-character type indicator (directory/catable/openable)
		if longList {
			var typeIndicator string
			if entry.IsDir() {
				typeIndicator = "d--"
				name = name + "/"
			} else {
				// All files are catable, determine if also openable
				// Files are openable if they contain a valid URL
				entryPath := targetPath + "/" + entry.Name()
				if targetPath == "" {
					entryPath = entry.Name()
				}
				isOpenable := extractURLFromContents(tfs, entryPath) != ""
				if isOpenable {
					typeIndicator = "-co"
				} else {
					typeIndicator = "-c-"
				}
			}
			output.WriteString(fmt.Sprintf("%s  %s", typeIndicator, name))
			continue
		}

		// Short format: just add / for directories
		if entry.IsDir() {
			name = name + "/"
		}
		output.WriteString(name)
	}

	return output.String(), nil
}

// PrintWorkingDirectory returns the current working directory path.
// Returns current working directory path and error. On success, returns (path, nil).
// This function typically does not error, but follows the same pattern for consistency.
func PrintWorkingDirectory(sessMgr SessionManager, sessionID string) (string, error) {
	currDir := sessMgr.GetCurrentDir(sessionID)
	if currDir == "" {
		return "/", nil
	}
	return "/" + currDir, nil
}

// CatFile reads and returns the contents of a file.
// Returns file content and error. On success, returns (content, nil).
// On error, returns (filename, error) where filename is the file the user
// attempted to access, allowing the caller to format contextual error messages.
// Possible errors: ErrMissingArgument, ErrFileNotFound, ErrIsDirectory, ErrAccessDenied.
func CatFile(tfs *termfs.FS, sessMgr SessionManager, sessionID string, args []string) (string, error) {
	if len(args) < 1 {
		return "", ErrMissingArgument
	}

	currDir := sessMgr.GetCurrentDir(sessionID)
	targetPath := resolvePath(currDir, args[0])

	openPath := targetPath
	if openPath == "" {
		openPath = "."
	}

	info, err := fs.Stat(tfs, openPath)
	if err != nil {
		return args[0], fmt.Errorf("stat file %q: %w", openPath, mapFSErr(err))
	}

	if info.IsDir() {
		return args[0], ErrIsDirectory
	}

	content, err := fs.ReadFile(tfs, openPath)
	if err != nil {
		return args[0], fmt.Errorf("read file %q: %w", openPath, mapFSErr(err))
	}

	return string(content), nil
}

// OpenFile opens a file and extracts its URL.
// Returns the URL and error. On success, returns (url, nil).
// On error, returns (filename, error) where filename is the file the user
// attempted to access, allowing the caller to format contextual error messages.
// Possible errors: ErrMissingArgument, ErrFileNotFound, ErrIsDirectory, ErrNotOpenable.
func OpenFile(tfs *termfs.FS, sessMgr SessionManager, sessionID string, args []string) (string, error) {
	if len(args) < 1 {
		return "", ErrMissingArgument
	}

	currDir := sessMgr.GetCurrentDir(sessionID)
	filename := args[0]
	targetPath := resolvePath(currDir, filename)

	openPath := targetPath
	if openPath == "" {
		openPath = "."
	}

	info, err := fs.Stat(tfs, openPath)
	if err != nil {
		return filename, fmt.Errorf("stat file %q: %w", openPath, mapFSErr(err))
	}

	if info.IsDir() {
		return filename, ErrIsDirectory
	}

	url := extractURLFromContents(tfs, openPath)
	if url == "" {
		return filename, ErrNotOpenable
	}

	return url, nil
}

// GeneratePrompt generates a terminal prompt based on the current directory.
func GeneratePrompt(currDir string) string {
	switch currDir {
	case "home/guest":
		return "guest@machine:~$ "
	case "":
		return "guest@machine:/$ "
	default:
		return fmt.Sprintf("guest@machine:/%s$ ", strings.TrimPrefix(currDir, "/"))
	}
}

// mapFSErr maps filesystem errors to our domain-specific errors.
func mapFSErr(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, fs.ErrNotExist) {
		return ErrFileNotFound
	}
	if errors.Is(err, fs.ErrPermission) {
		return ErrAccessDenied
	}
	if errors.Is(err, fs.ErrInvalid) {
		return ErrAccessDenied
	}

	// For unknown errors, return access denied to avoid leaking internal details
	return ErrAccessDenied
}

// resolvePath resolves a target path relative to the current directory.
func resolvePath(currDir, targetPath string) string {
	switch targetPath {
	case "~", "$HOME":
		return "home/guest"
	case ".":
		return currDir
	}

	isAbsPath := strings.HasPrefix(targetPath, "/")
	if isAbsPath {
		cleaned := path.Clean(targetPath)
		if cleaned == "/" {
			return ""
		}
		return strings.TrimPrefix(cleaned, "/")
	}

	var fullPath string
	if currDir == "" {
		fullPath = "/" + targetPath
	} else {
		fullPath = "/" + currDir + "/" + targetPath
	}

	cleaned := path.Clean(fullPath)
	if cleaned == "/" {
		return ""
	}

	return strings.TrimPrefix(cleaned, "/")
}

// isValidPathArgument validates that an argument looks like a reasonable path
// and not like other types of arguments that might be added later.
func isValidPathArgument(arg string) bool {
	if arg == "" {
		return false
	}

	if len(arg) > 255 { // typical filesystem path limit
		return false
	}

	if strings.Contains(arg, "\n") || strings.Contains(arg, "\r") {
		return false
	}

	if strings.Contains(arg, "\x00") {
		return false
	}

	return true
}

// extractURLFromContents extracts the URL from the files contents.
// URLs are extracted from the first occurence of the **URL:** prefix.
func extractURLFromContents(tfs *termfs.FS, filePath string) string {
	info, err := fs.Stat(tfs, filePath)
	if err != nil || info.IsDir() {
		return ""
	}

	f, err := tfs.Open(filePath)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if after, ok := strings.CutPrefix(line, "**URL:**"); ok {
			url := strings.TrimSpace(after)
			if url != "" {
				return url
			}
		}
	}

	return ""
}
