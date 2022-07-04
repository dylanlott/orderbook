# orderbook
> an experimental order matching exchange engine in Go.

## Design

We have the `Account`, `Order`, and `Market` interfaces.

`Marketplace` is meant to be hooked up to a REST API.

```
Accounts |--> Orders -->|
```

# Price and different assets

A fundamental our exchange will have to account for is the price of different assets.
Translating the different assets to a coherent standard that we can handle is
important to get right.

Our `Price` interface hopes to account for this translation.

# Market

The Market is the core interface of our exchange.

The `Marketplace` is a collection of `Markets`, the real core of our engine.

```golang
type `Marketplace` interface {
  List() []Market
  Add() Market
  Remove() error
}
```
