package herokucmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/stackerr"
)

type gitInfo struct {
	repo        string
	authToken   string
	remote      string
	description string
}

func (g *gitInfo) whichGit() error {
	_, err := exec.LookPath("git")
	if err != nil {
		return stackerr.New(`Unable to locate "git".
Please install "git" and ensure that you are able to run "git help" from the command prompt.`,
		)
	}
	return nil
}

func (g *gitInfo) isGitRepo(e *parsecli.Env) error {
	_, err := os.Lstat(filepath.Join(e.Root, ".git"))
	if os.IsNotExist(err) {
		return stackerr.Newf("%s is not a git repository. Please run 'git init'", e.Root)
	}
	return nil
}

func (g *gitInfo) checkAbort(e *parsecli.Env) error {
	err := g.whichGit()
	if err != nil {
		return err
	}
	return g.isGitRepo(e)
}

func (g *gitInfo) runCmd(cmd *exec.Cmd, msg string) error {
	err := cmd.Run()
	if err == nil {
		return nil
	}
	if _, ok := err.(*exec.ExitError); ok {
		if msg != "" {
			return stackerr.New(msg)
		}
		return stackerr.Newf("error executing: %q", strings.Join(cmd.Args, " "))
	}
	return stackerr.Wrap(err)
}

func (g *gitInfo) getDescription() string {
	if g.description != "" {
		return g.description
	}
	return fmt.Sprintf("committed by parse-cli at: %s", time.Now())
}

func (g *gitInfo) isDirty(e *parsecli.Env) (bool, error) {
	if err := g.checkAbort(e); err != nil {
		return false, err
	}

	cmd := exec.Command("git", "status", "--porcelain")
	var cmdOut, cmdErr bytes.Buffer
	cmd.Stdout = &cmdOut
	cmd.Stderr = &cmdErr
	err := g.runCmd(cmd, fmt.Sprintf("Unable to determine the status of git repo: %s", g.repo))
	if err != nil {
		fmt.Fprintln(e.Err, cmdErr.String())
		return false, err
	}

	if cmdOut.Len() == 0 {
		return false, nil
	}

	fmt.Fprintf(e.Out, "Status of git repo: %q:\n%s", g.repo, cmdOut.String())
	return true, nil
}

func (g *gitInfo) init(e *parsecli.Env) error {
	err := g.whichGit()
	if err != nil {
		return err
	}

	cmd := exec.Command("git", "init")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return g.runCmd(cmd, "")
}

func (g *gitInfo) commit(e *parsecli.Env) error {
	if err := g.checkAbort(e); err != nil {
		return err
	}

	cmd := exec.Command("git", "add", "-A", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := g.runCmd(cmd, fmt.Sprintf("error executing: %s", strings.Join(cmd.Args, " ")))
	if err != nil {
		return err
	}

	cmd = exec.Command("git", "commit", "-a", "-m", g.getDescription())
	return g.runCmd(cmd, fmt.Sprintf("error executing: %s", strings.Join(cmd.Args, " ")))
}

func (g *gitInfo) push(e *parsecli.Env, force bool) error {
	if err := g.checkAbort(e); err != nil {
		return err
	}
	command := []string{"push"}
	if force {
		command = append(command, "-f")
	}
	command = append(command,
		fmt.Sprintf("https://:%s@%s", g.authToken, g.remote),
		"master",
	)
	cmd := exec.Command("git", command...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return g.runCmd(
		cmd,
		fmt.Sprintf("Unable to push to git remote: %s", g.remote),
	)
}

func (g *gitInfo) isEmptyRepository(path string) (bool, error) {
	var files bytes.Buffer
	cmd := exec.Command("git", "-C", path, "ls-files")
	cmd.Stdout = &files
	cmd.Stderr = os.Stderr
	if err := g.runCmd(cmd, ""); err != nil {
		return false, err
	}
	return files.Len() == 0, nil
}

func (g *gitInfo) pull(e *parsecli.Env, gitURL string) error {
	if err := g.checkAbort(e); err != nil {
		return err
	}
	pullCmd := exec.Command("git", "pull", gitURL)
	pullCmd.Stdout = os.Stdout
	pullCmd.Stdin = os.Stdin
	return g.runCmd(pullCmd, "")
}

func (g *gitInfo) clone(gitURL, path string) error {
	_, err := exec.LookPath("git")
	if err != nil {
		return stackerr.New(`Unable to locate "git".
Please install "git" and ensure that you are able to run "git help" from the command prompt.`,
		)
	}

	cmd := exec.Command("git", "clone", gitURL, path)
	cmd.Stderr = os.Stderr
	return g.runCmd(cmd, fmt.Sprintf("error executing: %s", strings.Join(cmd.Args, " ")))
}
