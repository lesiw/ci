package main

import (
	"os"

	"lesiw.io/ci"
	"lesiw.io/cmdio"
	"lesiw.io/cmdio/cmd"
)

var targets = [][]string{
	{"linux", "386"},
	{"linux", "amd64"},
	{"linux", "arm"},
	{"linux", "arm64"},
	{"darwin", "amd64"},
	{"darwin", "arm64"},
	{"windows", "386"},
	{"windows", "arm"},
	{"windows", "amd64"},
	{"plan9", "386"},
	{"plan9", "arm"},
	{"plan9", "amd64"},
}

type actions struct{}

var project = new(actions)

func main() {
	defer cmdio.Recover(os.Stderr)
	args := os.Args[1:]
	if len(args) == 0 {
		args = append(args, "build")
	}
	ci.ActionHandler(project, args...)
}

func (a actions) Build() {
	a.Lint()
	a.Test()
	a.Race()
	for _, target := range targets {
		cmd.Env(map[string]string{
			"CGO_ENABLED": "0",
			"GOOS":        target[0],
			"GOARCH":      target[1],
		}).MustRun("go", "build", "-o", "/dev/null")
	}
}

func (a actions) Lint() {
	ensureGolangci()
	cmd.MustRun("golangci-lint", "run")

	cmd.MustRun("go", "run", "github.com/bobg/mingo/cmd/mingo@latest", "-check")
}

func ensureGolangci() {
	if cmd.MustCheck("which", "golangci-lint").Ok {
		return
	}
	gopath := cmd.MustGet("go", "env", "GOPATH")
	cmdio.MustRunPipe(
		cmd.Command("curl", "-sSfL",
			"https://raw.githubusercontent.com/golangci"+
				"/golangci-lint/master/install.sh"),
		cmd.Command("sh", "-s", "--", "-b", gopath.Output+"/bin"),
	)
}

func (a actions) Test() {
	ensureGoTestSum()
	cmd.MustRun("gotestsum", "./...")
}

func ensureGoTestSum() {
	if cmd.MustCheck("which", "gotestsum").Ok {
		return
	}
	cmd.MustRun("go", "install", "gotest.tools/gotestsum@latest")
}

func (a actions) Race() {
	cmd.MustRun("go", "build", "-race", "-o", "/dev/null")

}

func (a actions) Bump() {
	versionfile := "cmd/ci/version.txt"
	bump := cmdio.MustGetPipe(
		cmd.Command("curl", "lesiw.io/bump"),
		cmd.Command("sh"),
	).Output
	version := cmdio.MustGetPipe(
		cmd.Command("cat", versionfile),
		cmd.Command(bump, "-s", "1"),
		cmd.Command("tee", versionfile),
	).Output
	cmd.MustRun("git", "add", versionfile)
	cmd.MustRun("git", "commit", "-m", version)
	cmd.MustRun("git", "tag", version)
	cmd.MustRun("git", "push")
	cmd.MustRun("git", "push", "--tags")
}
