package main

import (
	"testing"

	"github.com/facebookgo/ensure"
)

func TestCmdScoreLess(t *testing.T) {
	t.Parallel()
	scores := cmdScores([]cmdScore{
		{levenshtein: 1, jaroWinkler: 0.9},
		{levenshtein: 2, jaroWinkler: 0.8},
	})

	ensure.True(t, scores.Less(0, 1))

	scores = cmdScores([]cmdScore{
		{levenshtein: 1, jaroWinkler: 0.9},
		{levenshtein: 1, jaroWinkler: 0.8},
	})

	ensure.True(t, scores.Less(0, 1))

	scores = cmdScores([]cmdScore{
		{levenshtein: 1, jaroWinkler: 0.8},
		{levenshtein: 1, jaroWinkler: 0.9},
	})

	ensure.False(t, scores.Less(0, 1))

	scores = cmdScores([]cmdScore{
		{levenshtein: 2, jaroWinkler: 0.9},
		{levenshtein: 1, jaroWinkler: 0.8},
	})

	ensure.False(t, scores.Less(0, 1))
}

func TestmakeCorrections(t *testing.T) {
	t.Parallel()

	ensure.DeepEqual(t, makeCorrections(nil, nil), "")
	ensure.DeepEqual(t, makeCorrections(nil, []string{"arg"}), "")

	subCommands := []string{"cmd"}
	ensure.DeepEqual(t, makeCorrections(subCommands, nil), "")

	ensure.DeepEqual(t, makeCorrections(subCommands, []string{"--file"}), "")
	ensure.DeepEqual(t, makeCorrections(subCommands, []string{"-f"}), "")
	ensure.DeepEqual(t, makeCorrections(subCommands, []string{"--file", "-p"}), "")
	ensure.DeepEqual(t, makeCorrections(subCommands, []string{"-p", "--file"}), "")

	args := []string{"--flags", "cmd", "args"}
	ensure.DeepEqual(t, makeCorrections(subCommands, args), "")
	ensure.DeepEqual(t, args, []string{"--flags", "cmd", "args"})
}

func TestMatchesForSubCommands(t *testing.T) {
	t.Parallel()

	subCommands := []string{"version", "deploy", "deplore"}

	testCases := []struct {
		args    []string
		modArgs []string
		message string
	}{
		{
			[]string{"version"},
			[]string{"version"},
			"",
		},
		{
			[]string{"vers"},
			[]string{"version"},
			"(assuming by `vers` you meant `version`)",
		},
		{
			[]string{"depluy", "-f", "-v"},
			[]string{"deploy", "-f", "-v"},
			"(assuming by `depluy` you meant `deploy`)",
		},
		{
			[]string{"-v", "deple", "-f"},
			[]string{"-v", "deple", "-f"},
			`ambiguous subcommand:	did you mean one of:
+ deploy
+ deplore
`,
		},
	}

	for _, testCase := range testCases {
		ensure.DeepEqual(t, makeCorrections(subCommands, testCase.args), testCase.message)
		ensure.DeepEqual(t, testCase.args, testCase.modArgs)
	}

	errorCases := []struct {
		args     []string
		modeArgs []string
	}{
		{
			[]string{"clivers"},
			[]string{"version"},
		},
		{
			[]string{"dupiay"},
			[]string{"deploy"},
		},
	}
	for _, errorCase := range errorCases {
		ensure.DeepEqual(t, makeCorrections(subCommands, errorCase.args), "")
	}
}
