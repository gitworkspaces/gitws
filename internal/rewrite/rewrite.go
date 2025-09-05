package rewrite

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// RewriteURL rewrites a URL to use the SSH alias
func RewriteURL(input, alias string) (org, repo, sshURL string, err error) {
	// Handle ORG/REPO format
	if org, repo, ok := parseOrgRepo(input); ok {
		sshURL = fmt.Sprintf("git@%s:%s/%s.git", alias, org, repo)
		return org, repo, sshURL, nil
	}

	// Handle HTTPS URLs
	if org, repo, ok := parseHTTPSURL(input); ok {
		sshURL = fmt.Sprintf("git@%s:%s/%s.git", alias, org, repo)
		return org, repo, sshURL, nil
	}

	// Handle SSH URLs
	if org, repo, ok := parseSSHURL(input); ok {
		sshURL = fmt.Sprintf("git@%s:%s/%s.git", alias, org, repo)
		return org, repo, sshURL, nil
	}

	return "", "", "", fmt.Errorf("unable to parse URL: %s", input)
}

// parseOrgRepo parses ORG/REPO format
func parseOrgRepo(input string) (org, repo string, ok bool) {
	// Simple regex for ORG/REPO format
	re := regexp.MustCompile(`^([a-zA-Z0-9._-]+)/([a-zA-Z0-9._-]+)$`)
	matches := re.FindStringSubmatch(input)
	if len(matches) == 3 {
		return matches[1], matches[2], true
	}
	return "", "", false
}

// parseHTTPSURL parses HTTPS URLs
func parseHTTPSURL(input string) (org, repo string, ok bool) {
	u, err := url.Parse(input)
	if err != nil {
		return "", "", false
	}

	if u.Scheme != "https" {
		return "", "", false
	}

	// Extract path components
	path := strings.TrimPrefix(u.Path, "/")
	path = strings.TrimSuffix(path, ".git")

	parts := strings.Split(path, "/")
	if len(parts) >= 2 {
		return parts[0], parts[1], true
	}

	return "", "", false
}

// parseSSHURL parses SSH URLs
func parseSSHURL(input string) (org, repo string, ok bool) {
	// Handle git@host:org/repo.git format
	re := regexp.MustCompile(`^git@([^:]+):([^/]+)/([^/]+)(?:\.git)?$`)
	matches := re.FindStringSubmatch(input)
	if len(matches) == 4 {
		repo = strings.TrimSuffix(matches[3], ".git")
		return matches[2], repo, true
	}

	return "", "", false
}

// NormalizeRepoName normalizes a repository name by removing .git suffix
func NormalizeRepoName(repo string) string {
	return strings.TrimSuffix(repo, ".git")
}

// ExtractHostFromSSHURL extracts the host from an SSH URL
func ExtractHostFromSSHURL(sshURL string) (string, error) {
	re := regexp.MustCompile(`^git@([^:]+):`)
	matches := re.FindStringSubmatch(sshURL)
	if len(matches) == 2 {
		return matches[1], nil
	}
	return "", fmt.Errorf("unable to extract host from SSH URL: %s", sshURL)
}
