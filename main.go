package main

import (
	"log/slog"

	"github.com/kkx999/KomariX/cmd"
	"github.com/kkx999/KomariX/utils"
	logger "github.com/kkx999/KomariX/utils/log"
)

func main() {
	if utils.VersionHash == "unknown" {
		logger.Setup(slog.LevelDebug)
	} else {
		logger.Setup(slog.LevelInfo)
	}

	logger.Infof("server", "KomariX Monitor %s (hash: %s)", utils.CurrentVersion, utils.VersionHash)

	cmd.Execute()
}
