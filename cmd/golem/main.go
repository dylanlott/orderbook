package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dylanlott/orderbook/pkg/accounts"
	"github.com/dylanlott/orderbook/pkg/orderbook"
	"github.com/dylanlott/orderbook/pkg/server"
)

// LatestOrderbook holds the global state of the book as the
// latest view received from the engine.
var LatestOrderbook *orderbook.Book

func main() {
	rootCmd := &cobra.Command{
		Use:   "golem",
		Short: "an order matching engine in Go",
		RunE: func(cmd *cobra.Command, args []string) error {
			motd()

			// create a context
			ctx := context.Background()

			// setup an accounts manager
			accts := &accounts.InMemoryManager{}

			// setup channels for wrapping our market
			in := make(chan orderbook.Order)
			out := make(chan *orderbook.Match)
			status := make(chan []orderbook.Order)

			// Run the book
			go orderbook.Run(ctx, accts, in, out, status)

			// start the server to bolt up to the engine
			engine := server.NewServer(accts, in, out, status)

			// run the server
			return engine.Run()
		},
	}

	rootCmd.PersistentFlags().String("config", "", "config file (default is $HOME/.golem.yaml)")
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.SetDefault("config", "$HOME/.golem.yaml")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}

func motd() {
	fmt.Printf(`
===================================================
||  ██████   ██████  ██      ███████ ███    ███  ||
|| ██       ██    ██ ██      ██      ████  ████  ||
|| ██   ███ ██    ██ ██      █████   ██ ████ ██  ||
|| ██    ██ ██    ██ ██      ██      ██  ██  ██  ||
||  ██████   ██████  ███████ ███████ ██      ██  ||
===================================================
`)
}
