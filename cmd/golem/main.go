package main

import (
	"context"
	"fmt"

	"github.com/dylanlott/orderbook/pkg/orderbook"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// LatestOrderbook holds the global state of the book as the
// latest view received from the engine.
var LatestOrderbook *orderbook.Book

func main() {
	rootCmd := &cobra.Command{
		Use:   "golem",
		Short: "an order matching engine in Go",
		Run: func(cmd *cobra.Command, args []string) {
			motd()
			ctx := context.Background()
			orderbook.Run(ctx)
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
