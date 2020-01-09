package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/joncherry/blockchain-miniproject/cmd/internal/resources"
)

func main() {
	app := &cli.App{
		Name:  "blockchain mini",
		Usage: "Handle and make requests to the network as a full node",
		Flags: []cli.Flag{
			&cli.Int64Flag{
				Name:    "max-transactions",
				Usage:   "The maximum number of transactions per block",
				Value:   500,
				EnvVars: []string{"MAX_TRANSACTIONS"},
			},
			&cli.Int64Flag{
				Name:    "time-limit",
				Usage:   "The time limit for adding transactions to each block in minutes",
				Value:   10,
				EnvVars: []string{"TIME_LIMIT"},
			},
			&cli.StringFlag{
				Name:    "host",
				Usage:   "The host endpoint of the node (please include the port)",
				EnvVars: []string{"HOST"},
			},
			&cli.StringFlag{
				Name:    "blockchain-folder-name",
				Usage:   "The folder that the blockchain file(s) will be written to",
				Value:   "written",
				EnvVars: []string{"BLOCKCHAIN_FOLDER_NAME"},
			},
		},
		Action: resources.Serve,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
