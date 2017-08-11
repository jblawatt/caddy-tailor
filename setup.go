package tailor

import (
	"github.com/mholt/caddy"
	"github.com/mholt/caddy/caddyhttp/httpserver"
)

func setup(c *caddy.Controller) error {
	cfg := httpserver.GetConfig(c)

	cfg.AddMiddleware(func(next httpserver.Handler) httpserver.Handler {
		return &Tailor{Next: next}
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
