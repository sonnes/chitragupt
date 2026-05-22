package main

import (
	"context"
	"os"

	"github.com/charmbracelet/log"
	"github.com/urfave/cli/v3"
)

func main() {
	root := &cli.Command{
		Name:      "cg",
		Usage:     "Generate readable transcripts from CLI agent session logs",
		UsageText: "cg <command> [options]",
		Description: `
     _     _ _                            _
  __| |__ (_) |_ _ _ __ _ __ _ _  _ _ __ | |_
 / _| '_ \| |  _| '_/ _' / _' | || | '_ \  _|
 \__|_.__/_|_|\__|_| \__,_\__, |\_,_| .__/\__|
                          |___/     |_|

 Start with:
   cg render --agent claude --file session.jsonl

 More detail:
   cg render --help   Generate terminal, HTML, or Markdown output.
   cg serve --help    Browse sessions locally without writing files.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "log",
				Usage:       "Log level: debug, info, warn, error",
				Value:       "error",
				DefaultText: "error",
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
		},
	}

	if err := root.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
