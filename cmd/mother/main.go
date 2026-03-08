package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/maxdml/mother/api"
	"github.com/maxdml/mother/internal/coder"
	"github.com/maxdml/mother/internal/workflow"

	"github.com/dbos-inc/dbos-transact-golang/dbos"
)

const listenAddr = ":8080"

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Initialize coder engine (replaces subprocess execution)
	workflow.CoderEngine = coder.New()

	dbosCtx, err := dbos.NewDBOSContext(context.Background(), dbos.Config{
		DatabaseURL:     databaseURL,
		AppName:         "mother",
		ConductorAPIKey: os.Getenv("DBOS_CONDUCTOR_API_KEY"),
	})
	if err != nil {
		log.Fatalf("failed to initialize DBOS: %v", err)
	}

	dbos.RegisterWorkflow(dbosCtx, workflow.CoderWorkflow)

	if err := dbos.Launch(dbosCtx); err != nil {
		log.Fatalf("failed to launch DBOS: %v", err)
	}
	defer dbos.Shutdown(dbosCtx, 5*time.Second)

	jobs := workflow.NewDBOSJobManager(dbosCtx)
	handler := &api.APIHandler{Jobs: jobs}

	mux := api.Handler(handler)

	log.Printf("mother listening on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
