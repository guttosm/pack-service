// Package main is the entry point for the pack-service application.
//
// @title           Pack Service API
// @version         1.0.0
// @description     API for calculating optimal pack combinations to fulfill orders.
//
//	This service determines the most efficient way to pack items using available pack sizes.
//
// @termsOfService  http://swagger.io/terms/
//
// @contact.name   API Support
// @contact.email  support@example.com
// @contact.url    https://github.com/guttosm/pack-service
//
// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT
//
// @host      localhost:8080
// @BasePath  /
//
// @securityDefinitions.apikey  ApiKeyAuth
// @in                          header
// @name                        X-API-Key
// @description                 API key for authentication. Required if authentication is enabled.
//
// @tag.name        Packs
// @tag.description Pack calculation operations
//
// @tag.name        Auth
// @tag.description Authentication and authorization endpoints
//
// @tag.name        Health
// @tag.description Health check endpoints
package main

import (
	_ "github.com/guttosm/pack-service/docs" // swagger docs

	"github.com/guttosm/pack-service/config"
	"github.com/guttosm/pack-service/internal/app"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg := config.Load()

	router := app.InitializeApp(cfg)
	server := app.NewServer(router, cfg.Server.Port)

	if err := server.Run(); err != nil {
		log.Fatal().Err(err).Msg("Server error")
	}
}
