package main

import (
	"fmt"
	"os"

	"agent-skills/internal/cli"
)

func main() {
	if err := cli.Run(os.Args, cli.Options{CommandName: "skill-installer"}); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
