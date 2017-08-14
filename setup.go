package tailor

import (
	"github.com/mholt/caddy"
	"github.com/mholt/caddy/caddyhttp/httpserver"
	"time"
)

// TailorConfig is the container for default configuration
type TailorConfig struct {
	DefaultTimeout    time.Duration
	ShowFragmentError bool
}

func parseConfig(c *caddy.Controller) TailorConfig {
	// for c.Next() {
	// 	args := c.RemainingArgs()
	// 	val := c.Val()

	// }
	return TailorConfig{
		DefaultTimeout:    1 * time.Minute,
		ShowFragmentError: true,
	}
}

func setup(c *caddy.Controller) error {
	cfg := httpserver.GetConfig(c)
	config := parseConfig(c)
	cfg.AddMiddleware(func(next httpserver.Handler) httpserver.Handler {
		return &Tailor{Next: next, Config: config}
	})

	return nil
}

func init() {

	caddy.RegisterPlugin("tailor", caddy.Plugin{
		ServerType: "http",
		Action:     setup,
	})
	httpserver.RegisterDevDirective("tailor", "jwt")
}
