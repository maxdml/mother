package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"mother/control-plane/api"

	"github.com/dbos-inc/dbos-transact-golang/dbos"
)

const (
	listenAddr      = ":8080"
	coderBinaryPath = "/Users/mother/mother/coder/coder"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	coderSvc = NewCoderService(coderBinaryPath)

	dbosCtx, err := dbos.NewDBOSContext(context.Background(), dbos.Config{
		DatabaseURL: databaseURL,
		AppName:     "control-plane",
	})
	if err != nil {
		log.Fatalf("failed to initialize DBOS: %v", err)
	}

	dbos.RegisterWorkflow(dbosCtx, CoderWorkflow)

	if err := dbos.Launch(dbosCtx); err != nil {
		log.Fatalf("failed to launch DBOS: %v", err)
	}
	defer dbos.Shutdown(dbosCtx, 5*time.Second)

	jobs := NewDBOSJobManager(dbosCtx)
	handler := &Handler{Jobs: jobs}

	mux := api.Handler(handler)

	log.Printf("control-plane listening on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
