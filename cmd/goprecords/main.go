package main

import (
	"os"

	"github.com/snonux/goprecords/internal/cli"
)

func main() {
	if err := cli.Execute(os.Args[1:]); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
