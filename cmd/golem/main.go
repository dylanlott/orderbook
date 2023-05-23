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
			ctx := context.Background()
			accts := &accounts.InMemoryManager{}
			in := make(chan orderbook.Order)
			out := make(chan *orderbook.Match)
			status := make(chan []orderbook.Order)

			go orderbook.Run(ctx, accts, in, out, status)

			engine := server.NewServer(accts, in, out, status)
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
									$$\                         
									$$ |                        
				 $$$$$$\   $$$$$$\  $$ | $$$$$$\  $$$$$$\$$$$\  
				$$  __$$\ $$  __$$\ $$ |$$  __$$\ $$  _$$  _$$\ 
				$$ /  $$ |$$ /  $$ |$$ |$$$$$$$$ |$$ / $$ / $$ |
				$$ |  $$ |$$ |  $$ |$$ |$$   ____|$$ | $$ | $$ |
				\$$$$$$$ |\$$$$$$  |$$ |\$$$$$$$\ $$ | $$ | $$ |
				 \____$$ | \______/ \__| \_______|\__| \__| \__|
				$$\   $$ |                                      
				\$$$$$$  |                                      
				 \______/                                       
			`)
}
