# orderbook

This package manages concurrent access to its buy and sell order lists, aka the orderbook.

This package started as a direct copy of `pkg/v5` but this will be the continued implementation now, since our initial exploration is finished.

## Order fills

Order fills are carried out in the attemptFill function. The hard problem of order filling is proper price traversal. This package uses a binary tree to solve that problem.

`Node` is the main type used by binary tree. It populates a binary tree for easy traversal of prices in an up and down fashion. Each `Node` has a `Price` and a `[]*Order` list.

Matches are made by lining up the two buy and sell sides of the book and trying to fill as many items from the opposite side (the "book order") until the order being filled ("the fill order") is complete.
