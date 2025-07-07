package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/meghashyamc/wheresthat/api"
)

func main() {
	godotenv.Load()
	ctx := context.Background()
	if err := api.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
