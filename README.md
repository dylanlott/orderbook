# orderbook
> an experimental order matching exchange engine in Go.

## Development

`make test` runs the test suite. 
Five iterations of this app are located in the `pkg/` directory, named `v1` through `v5` respectively.
The final implementation that applies the design points learned from the previous iterations lives in `pkg/orderbook`.

## Architecture

The two main packages are `accounts` and `orderbook`. Accounts holds an interface and an in-memory adapter for testing and use by other modules. Persistence via some KV store is on the roadmap for this project.

Orders are handled in the following process

1. OpWrites feed an order into the orderbook. 
2. The book inserts it into the tree and calls attemptFill on it in a goroutine. 
3. It generates matches until it's filled. Matches are fed into the Match channel.
4. The match channel processes the payment (buy and sell side) and passes it on the fill channel.

The fills channel is the only way to receive an update on an order. The orderbook is intentionally abstracts away the actual books, both sell and buy side, such that nothing above it can access or change those values. 

### Persistence

Bolt or BadgerDB are being explored currently for storing orders in a simple on-disk format without prematurely introducing a heavy database like Postgres.

## Golem CLI

Golem is the CLI client written in Viper that starts the orderbook. Located in `cmd/golem` it currently has only the root command which starts the server. Viper handles the configuration of the application by loading in the `-config` file path as well as a `$HOME/.golem.yml` config file.

