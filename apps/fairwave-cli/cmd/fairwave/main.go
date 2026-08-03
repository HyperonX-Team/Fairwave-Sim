// fairwave is the Fairwave operator CLI: nodes, SIMs, peers, spectrum gates.
package main

import (
	"fmt"
	"os"

	"github.com/HyperonX-Team/Fairwave-Sim/apps/fairwave-cli/internal/cli"
)

func main() {
	if err := cli.Root().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
