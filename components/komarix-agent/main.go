package main

import (
	"os"

	"github.com/kkx999/KomariX/components/komarix-agent/cmd"
)

func main() {
	cmd.Execute()
	os.Exit(0)
}
