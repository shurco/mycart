package middleware

import (
	"github.com/gofiber/contrib/v3/zerolog"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	recoverer "github.com/gofiber/fiber/v3/middleware/recover"
	gzerolog "github.com/rs/zerolog"
)

func Fiber(a *fiber.App, log *gzerolog.Logger) {
	a.Use(cors.New())
	a.Use(helmet.New())
	a.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))
	a.Use(zerolog.New(zerolog.Config{
		Logger: log,
	}))
	a.Use(recoverer.New())
}
