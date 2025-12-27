package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/zorcal/its-a-me-zorcal/internal/termfs"
	"github.com/zorcal/its-a-me-zorcal/internal/termui"
	"github.com/zorcal/its-a-me-zorcal/pkg/httprouter"
	"github.com/zorcal/its-a-me-zorcal/pkg/session"
)

type cmdTmplData struct {
	Command    string
	Output     template.HTML
	Error      bool
	Prompt     string
	NextPrompt string
}

func commandHandler(sessAdapter *sessionAdapter, tfs *termfs.FS) httprouter.Handler {
	tmpl, err := template.ParseFS(templatesFS, "templates/command_output.html")
	if err != nil {
		return func(w http.ResponseWriter, r *http.Request) error {
			return fmt.Errorf("parse template fs for command handler: %w", err)
		}
	}

	return func(w http.ResponseWriter, r *http.Request) error {
		if err := r.ParseForm(); err != nil {
			return wrapHTTPError(http.StatusBadRequest, "Bad form data", err)
		}

		sessionID := getSessionID(r)
		sess := sessAdapter.mgr.GetOrCreateSession(sessionID)

		// Handle any pending newlines first (sent as a parameter)
		if newlinesStr := r.FormValue("newlines"); newlinesStr != "" {
			if count, err := strconv.Atoi(newlinesStr); err == nil && count > 0 {
				currDir := sessAdapter.GetCurrentDir(sessionID)
				currPrompt := termui.GeneratePrompt(currDir)
				for range count {
					entry := newTerminalSessionEntry("", "", false)
					entry.Prompt = currPrompt
					sess.AddEntry(entry)
				}
			}
		}

		cmdLine := strings.TrimSpace(r.FormValue("command"))

		// Capture current directory and prompt before executing any command.
		currDir := sessAdapter.GetCurrentDir(sessionID)
		currPrompt := termui.GeneratePrompt(currDir)

		parts := strings.Fields(cmdLine)
		var cmd string
		var args []string

		if len(parts) > 0 {
			cmd = parts[0]
			args = parts[1:]
		}

		switch cmd {
		case "clear":
			runClearCommand(w, sess)
			return nil
		case "help":
			return runHelpCommand(w, sess, tmpl, cmdLine, currPrompt)
		case "open":
			return runOpenCommand(w, sess, tmpl, tfs, sessAdapter, sessionID, cmdLine, args, currPrompt)
		case "cd":
			return runCdCommand(w, sess, tmpl, tfs, sessAdapter, sessionID, cmdLine, args, currPrompt)
		case "ls":
			return runLsCommand(w, sess, tmpl, tfs, sessAdapter, sessionID, cmdLine, args, currPrompt)
		case "pwd":
			return runPwdCommand(w, sess, tmpl, sessAdapter, sessionID, currPrompt)
		case "cat":
			return runCatCommand(w, sess, tmpl, tfs, sessAdapter, sessionID, cmdLine, args, currPrompt)
		default:
			return runUnknownCommand(w, sess, tmpl, cmd, currPrompt)
		}
	}
}

func runClearCommand(w http.ResponseWriter, sess *session.Session[terminalSessionEntry]) {
	sess.ClearHistory()
	w.Header().Set("HX-Retarget", "#command-output")
	w.Header().Set("HX-Reswap", "innerHTML")
	w.Write([]byte(""))
}

func runHelpCommand(w http.ResponseWriter, sess *session.Session[terminalSessionEntry], tmpl *template.Template, cmdLine string, currPrompt string) error {
	output := `<div class="help">Available commands:

  <strong>ls [options] [path]</strong> - List directory contents
                        -a, --all: show hidden files (starting with .)
                        -l, --long: long format (d/c/o):
                            d-- = directory
                            -c- = catable
                            --o = openable
  <strong>cd [path]</strong>     - Change directory
  <strong>pwd</strong>           - Print working directory
  <strong>cat [file]</strong>    - Display file contents
  <strong>open [file]</strong>   - Open files containing URLs in browser
  <strong>clear</strong>         - Clear terminal history (or use Ctrl+L)
  <strong>help</strong>          - Show this help message

Notes:
  • Use Ctrl+L to clear the terminal

</div>`

	entry := newTerminalSessionEntry(cmdLine, template.HTML(output), false)
	entry.Prompt = currPrompt
	sess.AddEntry(entry)

	data := cmdTmplData{
		Command:    cmdLine,
		Output:     template.HTML(output),
		Error:      false,
		Prompt:     currPrompt,
		NextPrompt: currPrompt,
	}

	return tmpl.Execute(w, data)
}

func runOpenCommand(w http.ResponseWriter, sess *session.Session[terminalSessionEntry], tmpl *template.Template, tfs *termfs.FS, sessAdapter *sessionAdapter, sessionID, cmdLine string, args []string, currPrompt string) error {
	var (
		output  string
		isError bool
	)

	result, err := termui.OpenFile(tfs, sessAdapter, sessionID, args)
	if err != nil {
		isError = true
		switch {
		case errors.Is(err, termui.ErrMissingArgument):
			output = "open: missing file argument"
		case errors.Is(err, termui.ErrFileNotFound):
			output = fmt.Sprintf("open: %s: No such file or directory", result)
		case errors.Is(err, termui.ErrIsDirectory):
			output = fmt.Sprintf("open: %s: Is a directory", result)
		case errors.Is(err, termui.ErrNotOpenable):
			output = "open: file is not openable"
		default:
			output = "open: internal error"
		}
	} else {
		w.Header().Set("X-Open-URL", result)
		output = fmt.Sprintf("Opening %s in browser...", args[0])
		isError = false
	}

	entry := newTerminalSessionEntry(cmdLine, template.HTML(output), isError)
	entry.Prompt = currPrompt
	sess.AddEntry(entry)

	data := cmdTmplData{
		Command:    cmdLine,
		Output:     template.HTML(output),
		Error:      isError,
		Prompt:     currPrompt,
		NextPrompt: currPrompt,
	}

	return tmpl.Execute(w, data)
}

func runCdCommand(w http.ResponseWriter, sess *session.Session[terminalSessionEntry], tmpl *template.Template, tfs *termfs.FS, sessAdapter *sessionAdapter, sessionID, cmdLine string, args []string, currPrompt string) error {
	var (
		output  string
		isError bool
	)

	target, err := termui.ChangeDirectory(tfs, sessAdapter, sessionID, args)
	if err != nil {
		isError = true
		switch {
		case errors.Is(err, termui.ErrFileNotFound):
			output = fmt.Sprintf("cd: %s: No such file or directory", target)
		case errors.Is(err, termui.ErrNotDirectory):
			output = fmt.Sprintf("cd: %s: Not a directory", target)
		case errors.Is(err, termui.ErrAccessDenied):
			output = fmt.Sprintf("cd: %s: Permission denied", target)
		default:
			output = fmt.Sprintf("cd: %s: internal error", target)
		}
	}

	entry := newTerminalSessionEntry(cmdLine, template.HTML(output), isError)
	entry.Prompt = currPrompt
	sess.AddEntry(entry)

	data := cmdTmplData{
		Command:    cmdLine,
		Output:     template.HTML(output),
		Error:      isError,
		Prompt:     currPrompt,
		NextPrompt: termui.GeneratePrompt(sessAdapter.GetCurrentDir(sessionID)),
	}

	return tmpl.Execute(w, data)
}

func runLsCommand(w http.ResponseWriter, sess *session.Session[terminalSessionEntry], tmpl *template.Template, tfs *termfs.FS, sessAdapter *sessionAdapter, sessionID, cmdLine string, args []string, currPrompt string) error {
	var (
		output  string
		isError bool
	)

	result, err := termui.ListDirectoryContents(tfs, sessAdapter, sessionID, args)
	if err != nil {
		isError = true
		switch {
		case errors.Is(err, termui.ErrFileNotFound):
			output = fmt.Sprintf("ls: %s: No such file or directory", result)
		case errors.Is(err, termui.ErrTooManyArguments):
			output = "ls: too many arguments"
		case errors.Is(err, termui.ErrAccessDenied):
			output = "ls: Permission denied"
		case errors.Is(err, termui.ErrInvalidFlag):
			output = "ls: invalid flag or option"
		default:
			output = "ls: internal error"
		}
	} else {
		output = fmt.Sprintf("<pre class=\"file-list\">%s</pre>", result)
	}

	entry := newTerminalSessionEntry(cmdLine, template.HTML(output), isError)
	entry.Prompt = currPrompt
	sess.AddEntry(entry)

	data := cmdTmplData{
		Command:    cmdLine,
		Output:     template.HTML(output),
		Error:      isError,
		Prompt:     currPrompt,
		NextPrompt: currPrompt,
	}

	return tmpl.Execute(w, data)
}

func runPwdCommand(w http.ResponseWriter, sess *session.Session[terminalSessionEntry], tmpl *template.Template, sessAdapter *sessionAdapter, sessionID, currPrompt string) error {
	var (
		output  string
		isError bool
	)

	result, err := termui.PrintWorkingDirectory(sessAdapter, sessionID)
	if err != nil {
		isError = true
		output = "pwd: internal error"
	} else {
		output = result
	}

	entry := newTerminalSessionEntry("pwd", template.HTML(output), isError)
	entry.Prompt = currPrompt
	sess.AddEntry(entry)

	data := cmdTmplData{
		Command:    "pwd",
		Output:     template.HTML(output),
		Error:      isError,
		Prompt:     currPrompt,
		NextPrompt: currPrompt,
	}

	return tmpl.Execute(w, data)
}

func runCatCommand(w http.ResponseWriter, sess *session.Session[terminalSessionEntry], tmpl *template.Template, tfs *termfs.FS, sessAdapter *sessionAdapter, sessionID, cmdLine string, args []string, currPrompt string) error {
	var (
		output  string
		isError bool
	)

	result, err := termui.CatFile(tfs, sessAdapter, sessionID, args)
	if err != nil {
		isError = true
		switch {
		case errors.Is(err, termui.ErrMissingArgument):
			output = "cat: missing file argument"
		case errors.Is(err, termui.ErrFileNotFound):
			output = fmt.Sprintf("cat: %s: No such file or directory", result)
		case errors.Is(err, termui.ErrIsDirectory):
			output = fmt.Sprintf("cat: %s: Is a directory", result)
		case errors.Is(err, termui.ErrAccessDenied):
			output = fmt.Sprintf("cat: %s: Permission denied", result)
		default:
			output = "cat: internal error"
		}
	} else {
		output = fmt.Sprintf("<pre class=\"file-content\">%s</pre>", result)
	}

	entry := newTerminalSessionEntry(cmdLine, template.HTML(output), isError)
	entry.Prompt = currPrompt
	sess.AddEntry(entry)

	data := cmdTmplData{
		Command:    cmdLine,
		Output:     template.HTML(output),
		Error:      isError,
		Prompt:     currPrompt,
		NextPrompt: currPrompt,
	}

	return tmpl.Execute(w, data)
}

func runUnknownCommand(w http.ResponseWriter, sess *session.Session[terminalSessionEntry], tmpl *template.Template, cmd, currPrompt string) error {
	output := fmt.Sprintf("shell: %s: command not found...", cmd)

	entry := newTerminalSessionEntry(cmd, template.HTML(output), true)
	entry.Prompt = currPrompt
	sess.AddEntry(entry)

	data := cmdTmplData{
		Command:    cmd,
		Output:     template.HTML(output),
		Error:      true,
		Prompt:     currPrompt,
		NextPrompt: currPrompt,
	}

	return tmpl.Execute(w, data)
}

func newlineHandler(sessAdapter *sessionAdapter) httprouter.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		if err := r.ParseForm(); err != nil {
			return wrapHTTPError(http.StatusBadRequest, "Bad form data", err)
		}

		sessionID := getSessionID(r)
		sess := sessAdapter.mgr.GetOrCreateSession(sessionID)

		// Get current prompt for the session
		currDir := sessAdapter.GetCurrentDir(sessionID)
		currPrompt := termui.GeneratePrompt(currDir)

		count := 1
		if countStr := r.FormValue("count"); countStr != "" {
			if parsedCount, err := strconv.Atoi(countStr); err == nil && parsedCount > 0 {
				count = parsedCount
			}
		}

		for range count {
			entry := newTerminalSessionEntry("", "", false)
			entry.Prompt = currPrompt
			sess.AddEntry(entry)
		}

		w.WriteHeader(http.StatusNoContent)

		return nil
	}
}

func historyHandler(sessMgr *session.Manager[terminalSessionEntry]) httprouter.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		sessionID := getSessionID(r)
		sess := sessMgr.GetOrCreateSession(sessionID)

		history := sess.History()

		var commands []string
		for _, entry := range history {
			if strings.TrimSpace(entry.Command) != "" {
				commands = append(commands, entry.Command)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(commands); err != nil {
			return fmt.Errorf("json encode command history: %w", err)
		}

		return nil
	}
}
