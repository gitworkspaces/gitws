package rewrite

import (
	"testing"
)

func TestRewriteURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		alias    string
		expected struct {
			org    string
			repo   string
			sshURL string
			hasErr bool
		}
	}{
		{
			name:  "ORG/REPO format",
			input: "microsoft/vscode",
			alias: "github-work",
			expected: struct {
				org    string
				repo   string
				sshURL string
				hasErr bool
			}{
				org:    "microsoft",
				repo:   "vscode",
				sshURL: "git@github-work:microsoft/vscode.git",
				hasErr: false,
			},
		},
		{
			name:  "HTTPS URL",
			input: "https://github.com/microsoft/vscode.git",
			alias: "github-work",
			expected: struct {
				org    string
				repo   string
				sshURL string
				hasErr bool
			}{
				org:    "microsoft",
				repo:   "vscode",
				sshURL: "git@github-work:microsoft/vscode.git",
				hasErr: false,
			},
		},
		{
			name:  "HTTPS URL without .git",
			input: "https://github.com/microsoft/vscode",
			alias: "github-work",
			expected: struct {
				org    string
				repo   string
				sshURL string
				hasErr bool
			}{
				org:    "microsoft",
				repo:   "vscode",
				sshURL: "git@github-work:microsoft/vscode.git",
				hasErr: false,
			},
		},
		{
			name:  "SSH URL",
			input: "git@github.com:microsoft/vscode.git",
			alias: "github-work",
			expected: struct {
				org    string
				repo   string
				sshURL string
				hasErr bool
			}{
				org:    "microsoft",
				repo:   "vscode",
				sshURL: "git@github-work:microsoft/vscode.git",
				hasErr: false,
			},
		},
		{
			name:  "SSH URL without .git",
			input: "git@github.com:microsoft/vscode",
			alias: "github-work",
			expected: struct {
				org    string
				repo   string
				sshURL string
				hasErr bool
			}{
				org:    "microsoft",
				repo:   "vscode",
				sshURL: "git@github-work:microsoft/vscode.git",
				hasErr: false,
			},
		},
		{
			name:  "GitLab HTTPS URL",
			input: "https://gitlab.com/gitlab-org/gitlab.git",
			alias: "gitlab-work",
			expected: struct {
				org    string
				repo   string
				sshURL string
				hasErr bool
			}{
				org:    "gitlab-org",
				repo:   "gitlab",
				sshURL: "git@gitlab-work:gitlab-org/gitlab.git",
				hasErr: false,
			},
		},
		{
			name:  "Invalid URL",
			input: "not-a-url",
			alias: "github-work",
			expected: struct {
				org    string
				repo   string
				sshURL string
				hasErr bool
			}{
				org:    "",
				repo:   "",
				sshURL: "",
				hasErr: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org, repo, sshURL, err := RewriteURL(tt.input, tt.alias)

			if tt.expected.hasErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if org != tt.expected.org {
				t.Errorf("expected org %q, got %q", tt.expected.org, org)
			}

			if repo != tt.expected.repo {
				t.Errorf("expected repo %q, got %q", tt.expected.repo, repo)
			}

			if sshURL != tt.expected.sshURL {
				t.Errorf("expected sshURL %q, got %q", tt.expected.sshURL, sshURL)
			}
		})
	}
}

func TestNormalizeRepoName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"vscode", "vscode"},
		{"vscode.git", "vscode"},
		{"my-repo.git", "my-repo"},
		{"my_repo", "my_repo"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeRepoName(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExtractHostFromSSHURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		hasErr   bool
	}{
		{"git@github.com:microsoft/vscode.git", "github.com", false},
		{"git@github-work:microsoft/vscode.git", "github-work", false},
		{"git@gitlab.com:gitlab-org/gitlab.git", "gitlab.com", false},
		{"not-an-ssh-url", "", true},
		{"https://github.com/microsoft/vscode.git", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ExtractHostFromSSHURL(tt.input)

			if tt.hasErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
