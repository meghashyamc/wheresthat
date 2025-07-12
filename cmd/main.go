package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/meghashyamc/wheresthat/api"
	"github.com/meghashyamc/wheresthat/config"
)

func main() {
	godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %s\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	if err := api.Run(ctx, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
