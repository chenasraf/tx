package tmux

import "strings"

// TransformCmdToTmuxKeys converts a command string to tmux send-keys format
// Spaces become " Space " and newlines become " Enter "
func TransformCmdToTmuxKeys(cmd string) string {
	if strings.TrimSpace(cmd) == "" {
		return ""
	}

	var result strings.Builder
	keyMap := map[rune]string{
		' ':  "Space",
		'\n': "Enter",
	}

	for _, char := range cmd {
		if replacement, ok := keyMap[char]; ok {
			result.WriteString(" ")
			result.WriteString(replacement)
			result.WriteString(" ")
		} else {
			result.WriteRune(char)
		}
	}

	return result.String()
}
