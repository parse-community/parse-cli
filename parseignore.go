package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/facebookgo/parseignore"
	"github.com/facebookgo/symwalk"
)

func ignoreErrors(errors []error, e *env) string {
	l := len(errors)

	var b bytes.Buffer
	for n, err := range errors {
		b.WriteString(errorString(e, err))
		if n != l-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

type legacyRulesMatcher struct{}

func (l legacyRulesMatcher) Match(path string, fi os.FileInfo) (parseignore.Decision, error) {
	parts := strings.Split(path, string(filepath.Separator))
	for _, part := range parts {
		if strings.HasPrefix(part, ".") ||
			strings.HasPrefix(part, "#") ||
			strings.HasSuffix(part, ".swp") ||
			strings.HasSuffix(part, "~") {
			return parseignore.Exclude, nil
		}
	}
	return parseignore.Pass, nil
}

func parseIgnoreMatcher(content []byte) (parseignore.Matcher, []error) {
	if content != nil {
		matcher, errors := parseignore.CompilePatterns(content)
		return parseignore.MultiMatcher(matcher, legacyRulesMatcher{}), errors
	}
	return legacyRulesMatcher{}, nil
}

// parseIgnoreWalk is a helper function user by the Parse CLI. It walks the given
// root path (traversing symbolic links if any) and calls walkFn only
// on files that were not ignored by the given Matcher.
// Note: It ignores any errors encountered during matching
func parseIgnoreWalk(matcher parseignore.Matcher,
	root string,
	walkFn filepath.WalkFunc) ([]error, error) {
	var errors []error
	ignoresWalkFn := func(path string, info os.FileInfo, err error) error {
		// if root==path relPath="." which is ignored by legacyRule
		// hence the special handling of root
		if err != nil || root == path {
			return walkFn(path, info, err)
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		exclude, err := matcher.Match(relPath, info)
		if err != nil {
			errors = append(errors, err)
			return nil
		}

		if exclude == parseignore.Exclude {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		return walkFn(path, info, err)
	}

	return errors, symwalk.Walk(root, ignoresWalkFn)
}
