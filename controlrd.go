package main

import (
	"log"
	"path/filepath"

	"github.com/alecthomas/kong"
	"github.com/cskr/pubsub"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/jbrodriguez/controlrd/daemon/cmd"
	"github.com/jbrodriguez/controlrd/daemon/domain"
)

var Version string

var cli struct {
	LogsDir string `default:"/var/log" help:"directory to store logs"`

	ShowUps bool `env:"SHOW_UPS" default:"false" help:"whether to provide ups status or not"`

	Boot cmd.Boot `cmd:"" default:"1" help:"start processing"`
}

func main() {
	ctx := kong.Parse(&cli)

	log.SetOutput(&lumberjack.Logger{
		Filename:   filepath.Join(cli.LogsDir, "controlrd.log"),
		MaxSize:    10, // megabytes
		MaxBackups: 10,
		MaxAge:     28, // days
		// Compress:   true, // disabled by default
	})

	err := ctx.Run(&domain.Context{
		Config: domain.Config{
			Version: Version,
			ShowUps: cli.ShowUps,
		},
		Hub: pubsub.New(623),
	})
	ctx.FatalIfErrorf(err)
}
