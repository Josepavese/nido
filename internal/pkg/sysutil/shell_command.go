package sysutil

import (
	"os"
	"strings"
)

// ShellCommand renders argv for the POSIX shell used by terminal launchers. It
// is for display/explicit shell transport; direct execution always uses argv.
func ShellCommand(argv []string) string {
	quoted := make([]string, len(argv))
	for i, arg := range argv {
		if arg != "" && !strings.ContainsAny(arg, " \t\r\n'\"\\$`;|&()<>*?[]{}!~") {
			quoted[i] = arg
			continue
		}
		quoted[i] = "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
	}
	return strings.Join(quoted, " ")
}
func NullDevice() string { return os.DevNull }
