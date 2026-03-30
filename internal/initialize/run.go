package initialize

import (
	"fmt"
	"log"
	di_container "shortlink/internal/di-container"
	"shortlink/internal/validations"
)

func Run() {
	if err := validations.InitValidator(); err != nil {
		fmt.Printf("validations error")
	}

	LoadConfig()

	container, err := di_container.NewContainer()
	if err != nil {
		panic(fmt.Errorf("failed to initialize DI container: %w", err))
	}

	r := container.SetupRouter()

	log.Fatal(r.Run(":4000"))
}
