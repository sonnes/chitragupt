package main

import (
	"context"
	"os"

	"github.com/charmbracelet/log"
	"github.com/urfave/cli/v3"
)

func main() {
	root := &cli.Command{
		Name:  "cg",
		Usage: "Convert CLI agent session logs into shareable, human-readable transcripts",
		Description: `
     _     _ _                            _
  __| |__ (_) |_ _ _ __ _ __ _ _  _ _ __ | |_
 / _| '_ \| |  _| '_/ _' / _' | || | '_ \  _|
 \__|_.__/_|_|\__|_| \__,_\__, |\_,_| .__/\__|
                          |___/     |_|

 The scribe of sessions — turning agent logs into readable transcripts.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "log",
				Usage: "Log level: debug, info, warn, error",
				Value: "error",
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			level, err := log.ParseLevel(cmd.String("log"))
			if err != nil {
				return ctx, err
			}
			log.SetLevel(level)
			return ctx, nil
		},
		Commands: []*cli.Command{
			renderCmd(),
			serveCmd(),
			installCmd(),
		},
	}

	if err := root.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
