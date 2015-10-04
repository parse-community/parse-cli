package parsecli

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/xrash/smetrics"
)

const (
	levenThreshold = 4
	jaroThreshold  = 0.70
)

type cmdScore struct {
	name        string
	levenshtein int
	jaroWinkler float64
}

type cmdScores []cmdScore

func (s cmdScores) Len() int {
	return len(s)
}

func (s cmdScores) Less(i, j int) bool {
	if s[i].levenshtein < s[j].levenshtein {
		return true
	}
	if s[i].levenshtein == s[j].levenshtein {
		return s[i].jaroWinkler > s[j].jaroWinkler
	}
	return false
}

func (s cmdScores) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func getArgPos(args []string) int {
	pos := -1
	for i, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			pos = i
			break
		}
	}
	return pos
}

// SuggestCommands can be used to match a given word
// which is very similar to one among the list of available commands.
func SuggestCommands(given string, available []string) []string {
	commandSet := make(map[string]struct{})
	for _, command := range available {
		commandSet[command] = struct{}{}
	}
	if len(commandSet) == 0 || given == "" {
		return nil
	}

	if _, ok := commandSet[given]; ok {
		return nil
	}

	var scores cmdScores

	for command := range commandSet {
		levenshtein := smetrics.WagnerFischer(command, given, 1, 1, 1)
		jaroWinkler := smetrics.JaroWinkler(command, given, 0.7, 4)

		if levenshtein > levenThreshold {
			continue
		}
		if jaroWinkler < jaroThreshold {
			continue
		}
		scores = append(
			scores,
			cmdScore{
				name:        command,
				levenshtein: levenshtein,
				jaroWinkler: jaroWinkler,
			},
		)
	}
	if len(scores) == 0 {
		return nil
	}
	sort.Sort(scores)

	levenshtein := scores[0].levenshtein

	var matches []string
	for _, score := range scores {
		if score.levenshtein != levenshtein {
			break
		}
		matches = append(matches, score.name)
	}

	return matches
}

// MakeCorrections can be used to automatically correct the incorrect command in input args
// and return either an action that was taken or a suggestions message.
// One can also provide a map of aliases and related conversions
func MakeCorrections(commands []string, args []string) string {
	pos := getArgPos(args)
	if pos == -1 {
		return ""
	}

	arg := args[pos]

	matches := SuggestCommands(arg, commands)
	if len(matches) == 0 {
		return ""
	}
	if len(matches) == 1 {
		args[pos] = matches[0]
		return fmt.Sprintf("(assuming by `%s` you meant `%s`)", arg, matches[0])
	}

	var buffer bytes.Buffer
	for _, match := range matches {
		buffer.WriteString("+ ")
		buffer.WriteString(match)
		buffer.WriteRune('\n')
	}
	return fmt.Sprintf("ambiguous subcommand:\tdid you mean one of:\n%s", buffer.String())
}
