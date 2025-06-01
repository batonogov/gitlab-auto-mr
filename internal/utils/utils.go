package utils

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode"
)

// ReadDescriptionFile reads the content of a description file
func ReadDescriptionFile(filePath string) (string, error) {
	if filePath == "" {
		return "", nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open description file '%s': %w", filePath, err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read description file '%s': %w", filePath, err)
	}

	return string(content), nil
}

// ExtractIssueNumber extracts issue number from branch name
// Expects branch names like "feature/#6" or "bugfix/#123"
func ExtractIssueNumber(branchName string) (string, error) {
	// Regular expression to match #issue-number pattern
	re := regexp.MustCompile(`#(\d+)`)
	matches := re.FindStringSubmatch(branchName)

	if len(matches) < 2 {
		return "", fmt.Errorf("no issue number found in branch name '%s' (expected format: feature/#6)", branchName)
	}

	return matches[1], nil
}

// GenerateMRTitle generates a merge request title based on configuration
func GenerateMRTitle(sourceBranch, commitPrefix, customTitle string, useIssueName bool) (string, error) {
	// If custom title is provided, use it with prefix
	if customTitle != "" {
		if commitPrefix != "" {
			return fmt.Sprintf("%s: %s", commitPrefix, customTitle), nil
		}
		return customTitle, nil
	}

	// If useIssueName is true, try to extract issue information
	if useIssueName {
		issueNumber, err := ExtractIssueNumber(sourceBranch)
		if err != nil {
			return "", fmt.Errorf("failed to extract issue name: %w", err)
		}

		title := fmt.Sprintf("Resolve #%s", issueNumber)
		if commitPrefix != "" {
			title = fmt.Sprintf("%s: %s", commitPrefix, title)
		}
		return title, nil
	}

	// Default title based on branch name
	branchTitle := strings.ReplaceAll(sourceBranch, "-", " ")
	branchTitle = strings.ReplaceAll(branchTitle, "_", " ")
	branchTitle = toTitleCase(branchTitle)

	if commitPrefix != "" {
		return fmt.Sprintf("%s: %s", commitPrefix, branchTitle), nil
	}

	return branchTitle, nil
}

// SanitizeBranchName removes invalid characters from branch names
func SanitizeBranchName(branchName string) string {
	// Remove common prefixes like "origin/" and "refs/heads/"
	// Use a loop to handle nested or repeated prefixes
	for {
		oldBranch := branchName
		branchName = strings.TrimPrefix(branchName, "origin/")
		branchName = strings.TrimPrefix(branchName, "refs/heads/")
		if branchName == oldBranch {
			break // No more changes
		}
	}

	return strings.TrimSpace(branchName)
}

// IsValidURL checks if a string is a valid URL
func IsValidURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

// FormatUserIDs formats a slice of user IDs as a comma-separated string
func FormatUserIDs(userIDs []int) string {
	if len(userIDs) == 0 {
		return ""
	}

	var strs []string
	for _, id := range userIDs {
		strs = append(strs, fmt.Sprintf("%d", id))
	}

	return strings.Join(strs, ", ")
}

// toTitleCase converts a string to title case (capitalizes first letter of each word)
func toTitleCase(s string) string {
	if s == "" {
		return s
	}

	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			runes := []rune(word)
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}

	return strings.Join(words, " ")
}
