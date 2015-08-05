package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/facebookgo/ensure"
	"github.com/facebookgo/testname"
)

func makeEmptyRoot(t *testing.T) string {
	prefix := fmt.Sprintf("%s-", testname.Get("parse-cli-"))
	root, err := ioutil.TempDir("", prefix)
	ensure.Nil(t, err)
	return root
}

func initProject(t *testing.T, root string) ([]string, []string) {
	dirs := []string{
		filepath.Join(root, "tester"),
		filepath.Join(root, "tester", "inside"),
		filepath.Join(root, "tester", "inside", "test"),
		filepath.Join(root, "tester", "inside", "tester"),
	}

	for _, dir := range dirs {
		ensure.Nil(t, os.MkdirAll(dir, 0755))
	}

	files := []string{
		filepath.Join(root, "tester", "test"),
		filepath.Join(root, "tester", "inside", "tester", "test"),
	}

	for _, file := range files {
		f, err := os.Create(file)
		ensure.Nil(t, err)
		defer f.Close()
	}

	return files, dirs
}

func TestPatternWalker(t *testing.T) {
	t.Parallel()

	root := makeEmptyRoot(t)
	defer os.RemoveAll(root)

	files, dirs := initProject(t, root)
	dirs = append(dirs, root)

	var visitedFiles, visitedDirs []string
	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			visitedDirs = append(visitedDirs, path)
			return nil
		}
		visitedFiles = append(visitedFiles, path)
		return nil
	}

	testCases := []struct {
		ignores                     string
		expectedFiles, expectedDirs []string
	}{
		{
			`
test/
!tester/test
  `,
			[]string{},
			[]string{
				filepath.Join(root, "tester", "inside", "test"),
			},
		},
		{
			`
test
!tester/test
  `,
			[]string{
				filepath.Join(root, "tester", "inside", "tester", "test"),
			},
			[]string{
				filepath.Join(root, "tester", "inside", "test"),
			},
		},
		{
			`
test*
  `,
			files,
			[]string{
				filepath.Join(root, "tester"),
				filepath.Join(root, "tester", "inside"),
				filepath.Join(root, "tester", "inside", "test"),
				filepath.Join(root, "tester", "inside", "tester"),
			},
		},
	}

	for _, testCase := range testCases {
		matcher, errors := parseIgnoreMatcher([]byte(testCase.ignores))
		ensure.DeepEqual(t, len(errors), 0)

		visitedFiles, visitedDirs = nil, nil
		errors, err := parseIgnoreWalk(matcher, root, walkFn)
		ensure.Nil(t, err)
		ensure.DeepEqual(t, len(errors), 0)

		var aggFiles, aggDirs []string
		aggFiles = append(aggFiles, testCase.expectedFiles...)
		aggFiles = append(aggFiles, visitedFiles...)

		aggDirs = append(aggDirs, testCase.expectedDirs...)
		aggDirs = append(aggDirs, visitedDirs...)

		ensure.SameElements(t, aggFiles, files)
		ensure.SameElements(t, aggDirs, dirs)
	}
}

func TestPatternWalkerSymLink(t *testing.T) {
	t.Parallel()

	root := makeEmptyRoot(t)
	defer os.RemoveAll(root)

	files, dirs := initProject(t, root)
	ensure.Nil(t, os.Symlink(filepath.Join(root, "tester"), filepath.Join(root, "link")))

	for i := 0; i < len(files); i++ {
		files[i] = strings.Replace(files[i], "tester", "link", 1)
	}
	for i := 0; i < len(dirs); i++ {
		dirs[i] = strings.Replace(dirs[i], "tester", "link", 1)
	}

	var visitedFiles, visitedDirs []string
	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			visitedDirs = append(visitedDirs, path)
			return nil
		}
		visitedFiles = append(visitedFiles, path)
		return nil
	}

	testCases := []struct {
		ignores                     string
		expectedFiles, expectedDirs []string
	}{
		{
			`
test/
!tester/test
  `,
			[]string{},
			[]string{
				filepath.Join(root, "link", "inside", "test"),
			},
		},
		{
			`
test
!tester/test
  `,
			[]string{
				filepath.Join(root, "link", "inside", "tester", "test"),
				filepath.Join(root, "link", "test"),
			},
			[]string{
				filepath.Join(root, "link", "inside", "test"),
			},
		},
		{
			`
test*
  `,
			[]string{
				filepath.Join(root, "link", "test"),
				filepath.Join(root, "link", "inside", "tester", "test"),
			},
			[]string{
				filepath.Join(root, "link", "inside", "test"),
				filepath.Join(root, "link", "inside", "tester"),
			},
		},
	}

	for _, testCase := range testCases {
		matcher, errs := parseIgnoreMatcher([]byte(testCase.ignores))
		ensure.DeepEqual(t, len(errs), 0)

		visitedFiles, visitedDirs = nil, nil
		errors, err := parseIgnoreWalk(matcher, filepath.Join(root, "link"), walkFn)
		ensure.Nil(t, err)
		ensure.DeepEqual(t, len(errors), 0)

		var aggFiles, aggDirs []string
		aggFiles = append(aggFiles, testCase.expectedFiles...)
		aggFiles = append(aggFiles, visitedFiles...)

		aggDirs = append(aggDirs, testCase.expectedDirs...)
		aggDirs = append(aggDirs, visitedDirs...)

		ensure.SameElements(t, aggFiles, files)
		ensure.SameElements(t, aggDirs, dirs)
	}
}
