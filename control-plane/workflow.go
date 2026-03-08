package main

import (
	"context"
	"fmt"

	"mother/control-plane/api"

	"github.com/dbos-inc/dbos-transact-golang/dbos"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// CoderWorkflowInput is the input to the coder workflow.
type CoderWorkflowInput struct {
	Service string          `json:"service"`
	Params  api.CoderParams `json:"params"`
}

// coderSvc is set during initialization and used by CoderWorkflow.
var coderSvc *CoderService

// CoderWorkflow is a DBOS durable workflow that runs a coder job.
func CoderWorkflow(ctx dbos.DBOSContext, input CoderWorkflowInput) (string, error) {
	wfID, err := dbos.GetWorkflowID(ctx)
	if err != nil {
		return "", fmt.Errorf("get workflow ID: %w", err)
	}
	jobID, err := uuid.Parse(wfID)
	if err != nil {
		return "", fmt.Errorf("parse workflow ID as UUID: %w", err)
	}

	result, err := dbos.RunAsStep(ctx, func(stepCtx context.Context) (string, error) {
		return coderSvc.Run(stepCtx, openapi_types.UUID(jobID), input.Params)
	}, dbos.WithStepName("run_coder"))

	return result, err
}

// JobManager abstracts job lifecycle operations.
type JobManager interface {
	StartJob(ctx context.Context, service string, params api.CoderParams) (openapi_types.UUID, error)
	GetJob(ctx context.Context, id openapi_types.UUID) (*api.Job, error)
}

// DBOSJobManager implements JobManager using DBOS durable workflows.
type DBOSJobManager struct {
	dbosCtx dbos.DBOSContext
}

// NewDBOSJobManager creates a new DBOSJobManager.
func NewDBOSJobManager(dbosCtx dbos.DBOSContext) *DBOSJobManager {
	return &DBOSJobManager{dbosCtx: dbosCtx}
}

// StartJob starts a DBOS workflow for the given service and params.
func (m *DBOSJobManager) StartJob(ctx context.Context, service string, params api.CoderParams) (openapi_types.UUID, error) {
	id := newUUID()
	_, err := dbos.RunWorkflow(m.dbosCtx, CoderWorkflow, CoderWorkflowInput{
		Service: service,
		Params:  params,
	}, dbos.WithWorkflowID(id.String()))
	if err != nil {
		return openapi_types.UUID{}, fmt.Errorf("start workflow: %w", err)
	}
	return openapi_types.UUID(id), nil
}

// GetJob retrieves workflow status and maps it to an api.Job.
// Returns (nil, nil) if the workflow does not exist.
func (m *DBOSJobManager) GetJob(ctx context.Context, id openapi_types.UUID) (*api.Job, error) {
	handle, err := dbos.RetrieveWorkflow[string](m.dbosCtx, id.String())
	if err != nil {
		return nil, nil
	}

	status, err := handle.GetStatus()
	if err != nil {
		return nil, nil
	}

	job := &api.Job{
		Id:        id,
		Service:   "coder",
		CreatedAt: status.CreatedAt,
	}

	switch status.Status {
	case "SUCCESS":
		job.Status = api.Completed
		if s, ok := status.Output.(string); ok {
			job.Result = &s
		}
		t := status.UpdatedAt
		job.CompletedAt = &t
	case "ERROR", "CANCELLED", "MAX_RECOVERY_ATTEMPTS_EXCEEDED":
		job.Status = api.Failed
		if status.Error != nil {
			errStr := status.Error.Error()
			job.Error = &errStr
		}
		t := status.UpdatedAt
		job.CompletedAt = &t
	case "ENQUEUED":
		job.Status = api.Pending
	default: // PENDING = workflow is executing
		job.Status = api.Running
	}

	return job, nil
}
