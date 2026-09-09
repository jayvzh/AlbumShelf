// main 是服务入口，只做三件事：config.Load() → app.New() → app.Run()。
package main

import (
	"log"

	"imageshelf/backend/internal/app"
	"imageshelf/backend/internal/config"
)

func main() {
	cfg := config.Load()

	a := app.New(cfg)

	if err := a.Run(); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}
