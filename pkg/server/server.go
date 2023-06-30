package server

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dylanlott/orderbook/pkg/accounts"
	"github.com/dylanlott/orderbook/pkg/orderbook"

	"github.com/VictoriaMetrics/metrics"
	"github.com/labstack/echo/v4"
)

var defaultPort = ":1323"
var interval = time.Millisecond * 500
var pushProcessMetrics = false

// Engine is a fully-plumbed orderbook and account system
// hooked up to an echo server with a metrics client plugged in.
type Engine struct {
	srv    *echo.Echo
	state  []*orderbook.Order
	in     chan *orderbook.Order
	out    chan *orderbook.Match
	status chan []*orderbook.Order
}

// NewServer returns a new server.Engine that wires together
// API requests to the orderbook.
// startingx
func NewServer(
	accounts accounts.AccountManager,
	in chan *orderbook.Order,
	out chan *orderbook.Match,
	status chan []*orderbook.Order,
) *Engine {
	engine := &Engine{
		in:     in,
		out:    out,
		status: status,
	}

	err := metrics.InitPush("http://localhost:8428/write", interval, `label="orderbook"`, pushProcessMetrics)
	if err != nil {
		log.Fatalf("failed to connect to metrics platform: %+v", err)
	}

	e := echo.New()

	e.Use(count)

	e.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"name":    "orderbook",
			"version": "0.1",
		})
	})

	GetOrders := func(c echo.Context) error {
		return c.JSON(http.StatusOK, engine.state)
	}

	InsertOrder := func(c echo.Context) error {
		o := new(orderbook.Order)
		if err := c.Bind(o); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		e.Logger.Infof("order received: %+v", o)
		engine.in <- o
		return nil
	}

	e.GET("/orders", GetOrders)
	e.POST("/orders", InsertOrder)

	engine.srv = e

	engine.srv.Logger.Debugf("server created")

	// handle state updates
	handleState(engine, status)

	return engine
}

// Run starts the engine at defaultPort
func (eng *Engine) Run() error {
	return eng.srv.Start(defaultPort)
}

// handleState updates the Engine's view of the Orderbooks
// so that it can be fetched by the server.
func handleState(e *Engine, status chan []*orderbook.Order) {
	go func(e *Engine, status chan []*orderbook.Order) {
		for stats := range status {
			e.state = stats
			e.srv.Logger.Debugf("state: %+v\n", e.state)
		}
	}(e, status)
}

func count(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		path := metrics.GetOrCreateCounter(fmt.Sprintf(`requests_total{path=%s}`, c.Path()))
		path.Inc()
		counter := metrics.GetOrCreateCounter(`request_total`)
		counter.Add(1)
		return next(c)
	}
}
