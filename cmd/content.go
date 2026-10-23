package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/output"
)

// resolveContent returns content from a direct value or file flag.
// If value is "-", reads from stdin.
func resolveContent(value, file string) (string, error) {
	if value != "" && file != "" {
		return "", fmt.Errorf("cannot use both --body and --body-file")
	}

	if value == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		return string(data), nil
	}

	if value != "" {
		return bean.UnescapeBody(value), nil
	}

	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("reading file: %w", err)
		}
		return string(data), nil
	}

	return "", nil
}

// resolveAppendContent returns the text for --body-append.
// If value is "-", reads from stdin and drops trailing newlines; otherwise
// escape sequences like \n are interpreted.
func resolveAppendContent(value string) (string, error) {
	if value == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		return strings.TrimRight(string(data), "\n"), nil
	}
	return bean.UnescapeBody(value), nil
}

// applyTags adds tags to a bean, returning an error if any tag is invalid.
func applyTags(b *bean.Bean, tags []string) error {
	for _, tag := range tags {
		if err := b.AddTag(tag); err != nil {
			return err
		}
	}
	return nil
}

// formatCycle formats a cycle path for display.
func formatCycle(path []string) string {
	return strings.Join(path, " → ")
}

// cmdError returns an appropriate error for JSON or text mode.
// Note: Use %v instead of %w for error arguments - wrapping is not preserved in JSON mode.
func cmdError(jsonMode bool, code string, format string, args ...any) error {
	if jsonMode {
		return output.Error(code, fmt.Sprintf(format, args...))
	}
	return fmt.Errorf(format, args...)
}

// mergeTags combines existing tags with additions and removals.
func mergeTags(existing, add, remove []string) []string {
	tags := make(map[string]bool)
	for _, t := range existing {
		tags[t] = true
	}
	for _, t := range add {
		tags[t] = true
	}
	for _, t := range remove {
		delete(tags, t)
	}
	result := make([]string, 0, len(tags))
	for t := range tags {
		result = append(result, t)
	}
	return result
}

// handleBeanQueryError provides consistent error handling for bean queries.
// When an ambiguous ID is encountered:
// - In JSON mode: returns all matching beans as a JSON array
// - In text mode: returns a formatted error message with IDs and titles
func handleBeanQueryError(err error, jsonMode bool, idPrefix string) error {
	if err == nil {
		return nil
	}

	// Check for ambiguous ID error
	if ambigErr, ok := beancore.IsAmbiguousIDError(err); ok {
		if jsonMode {
			// Return all matching beans as JSON array
			return output.SuccessMultiple(ambigErr.Matches)
		}
		// Return formatted error message
		return fmt.Errorf("%s", ambigErr.Error())
	}

	// Not found
	if errors.Is(err, beancore.ErrNotFound) {
		return cmdError(jsonMode, output.ErrNotFound, "bean not found: %s", idPrefix)
	}

	// Other error
	return cmdError(jsonMode, output.ErrFileError, "failed to find bean: %v", err)
}
