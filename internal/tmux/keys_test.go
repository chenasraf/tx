package tmux

import (
	"strings"
	"testing"
)

func TestTransformCmdToTmuxKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: "",
		},
		{
			name:     "simple command no spaces",
			input:    "ls",
			expected: "ls",
		},
		{
			name:     "command with space",
			input:    "ls -la",
			expected: "ls Space -la",
		},
		{
			name:     "command with multiple spaces",
			input:    "git commit -m test",
			expected: "git Space commit Space -m Space test",
		},
		{
			name:     "command with newline",
			input:    "echo hello\necho world",
			expected: "echo Space hello Enter echo Space world",
		},
		{
			name:     "npm start",
			input:    "npm start",
			expected: "npm Space start",
		},
		{
			name:     "complex command",
			input:    "cd ~/project && npm install",
			expected: "cd Space ~/project Space && Space npm Space install",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TransformCmdToTmuxKeys(tt.input)
			// Normalize spaces for comparison
			resultNorm := strings.Join(strings.Fields(result), " ")
			expectedNorm := strings.Join(strings.Fields(tt.expected), " ")

			if resultNorm != expectedNorm {
				t.Errorf("TransformCmdToTmuxKeys(%q) = %q, expected %q", tt.input, resultNorm, expectedNorm)
			}
		})
	}
}

func TestTransformCmdToTmuxKeys_SpacePadding(t *testing.T) {
	result := TransformCmdToTmuxKeys("a b")

	// Should have space padding around "Space"
	if !strings.Contains(result, " Space ") {
		t.Errorf("expected ' Space ' in result, got %q", result)
	}
}

func TestTransformCmdToTmuxKeys_PreservesSpecialChars(t *testing.T) {
	tests := []struct {
		input string
		check string
	}{
		{"ls | grep foo", "|"},
		{"echo $HOME", "$"},
		{"test && run", "&&"},
		{"cmd > file", ">"},
		{"cmd < file", "<"},
		{"echo 'hello'", "'"},
		{`echo "hello"`, `"`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := TransformCmdToTmuxKeys(tt.input)
			if !strings.Contains(result, tt.check) {
				t.Errorf("expected %q to contain %q, got %q", tt.input, tt.check, result)
			}
		})
	}
}
