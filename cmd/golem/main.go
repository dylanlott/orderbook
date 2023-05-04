package main

import (
	"context"
	"fmt"
	"log"

	"github.com/dylanlott/orderbook/pkg/accounts"
	"github.com/dylanlott/orderbook/pkg/orderbook"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var bufferSize = 10_000

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
			writes := make(chan orderbook.OpWrite, bufferSize)
			accts := &accounts.InMemoryManager{}
			errs := make(chan error, bufferSize)
			fills := make(chan orderbook.FillResult, bufferSize)

			go func() {
				for err := range errs {
					log.Printf("[error]: %+v", err) // TODO: log.Fatalf here?
				}
			}()

			go func() {
				for fill := range fills {
					fmt.Printf("[FILL]: %v\n", fill)
					// DEVLOG: Hook matches up to pkg/account transactions and
					// populate each order []Transaction field
					//
					// TODO update sell order & buyer appropriately
					// TODO remove finished orders from the tree
					// TODO send finished orders on output channel
				}
			}()

			// Start processes reads and writes to
			// produce matches and errors
			// DEVLOG: CURRENTLY RESULTS IN DEADLOCK -- WHY??
			orderbook.Start(ctx, accts, writes, fills, errs)
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
