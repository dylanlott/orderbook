package server

import (
	"net/http"

	"github.com/dylanlott/orderbook/pkg/accounts"
	"github.com/dylanlott/orderbook/pkg/orderbook"
	"github.com/labstack/echo/v4"
)

// defaultPort the app starts on
var defaultPort = ":1323"

type Engine struct {
	srv   *echo.Echo
	state []orderbook.Order
}

// NewServer returns a new server.Engine that wires together
// API requests to the orderbook.
// startingx
func NewServer(
	accounts accounts.AccountManager,
	in chan orderbook.Order,
	out chan *orderbook.Match,
	status chan []orderbook.Order,
) *Engine {
	engine := &Engine{}

	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})
	engine.srv = e

	engine.srv.Logger.Debugf("server created")

	// handle state updates
	handleState(engine, status)

	return engine
}

func (eng *Engine) Run() error {
	return eng.srv.Start(defaultPort)
}

// handleState updates the Engine's view of the Orderbooks
// so that it can be fetched by the server.
func handleState(e *Engine, status chan []orderbook.Order) {
	go func(e *Engine, status chan []orderbook.Order) {
		for stats := range status {
			e.state = stats
			e.srv.Logger.Debugf("state: %+v\n", e.state)
		}
	}(e, status)
}
