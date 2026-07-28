// mcsmcli is a command-line management tool for the MCSManager panel.
package main

import (
	"os"

	"mcsmcli/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
