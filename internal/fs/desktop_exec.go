package fs

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// LaunchApp opens paths with a specific desktop application.
func LaunchApp(app DesktopApp, paths []string) error {
	return launchDetached(app.Exec, paths)
}

// LaunchCommand runs an arbitrary user-supplied command with the selection.
func LaunchCommand(commandLine string, paths []string) error {
	return launchDetached(commandLine, paths)
}

func launchDetached(commandLine string, paths []string) error {
	argv := BuildExecArgs(commandLine, paths)
	if len(argv) == 0 {
		return os.ErrInvalid
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// BuildExecArgs expands XDG Exec field codes (%f %F %u %U %i %c %k) for a selection.
func BuildExecArgs(execLine string, paths []string) []string {
	fields := splitExecLine(execLine)
	uris := make([]string, 0, len(paths))
	for _, p := range paths {
		uris = append(uris, FileURI(p))
	}

	var out []string
	for _, f := range fields {
		switch f {
		case "%f":
			if len(paths) > 0 {
				out = append(out, paths[0])
			}
		case "%u":
			if len(uris) > 0 {
				out = append(out, uris[0])
			}
		case "%F":
			out = append(out, paths...)
		case "%U":
			out = append(out, uris...)
		case "%i":
			// Icon field: intentionally skipped.
		case "%c":
			out = append(out, "Vela Files")
		case "%k":
			out = append(out, "file:///")
		default:
			out = append(out, f)
		}
	}

	if len(paths) > 0 && !strings.Contains(execLine, "%") {
		out = append(out, paths...)
	}
	return out
}

// splitExecLine tokenises an Exec line respecting single/double quotes.
func splitExecLine(line string) []string {
	var tokens []string
	var sb strings.Builder
	inSingle, inDouble, escaped := false, false, false

	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case escaped:
			sb.WriteByte(c)
			escaped = false
		case c == '\\' && !inSingle:
			escaped = true
		case c == '\'' && !inDouble:
			inSingle = !inSingle
		case c == '"' && !inSingle:
			inDouble = !inDouble
		case (c == ' ' || c == '\t') && !inSingle && !inDouble:
			if sb.Len() > 0 {
				tokens = append(tokens, sb.String())
				sb.Reset()
			}
		default:
			sb.WriteByte(c)
		}
	}
	if sb.Len() > 0 {
		tokens = append(tokens, sb.String())
	}
	return tokens
}

func execError(tool string, out []byte, err error) error {
	msg := strings.TrimSpace(string(out))
	if msg == "" && err != nil {
		msg = err.Error()
	}
	return &ExecError{Tool: tool, Message: msg}
}

// ExecError wraps a failed external command with its stderr output.
type ExecError struct {
	Tool    string
	Message string
}

func (e *ExecError) Error() string {
	return e.Tool + ": " + e.Message
}
