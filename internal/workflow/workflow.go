package workflow

import (
	"context"
	"fmt"

	"github.com/maxdml/mother/api"
	"github.com/maxdml/mother/internal/coder"

	"github.com/dbos-inc/dbos-transact-golang/dbos"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// CoderWorkflowInput is the input to the coder workflow.
type CoderWorkflowInput struct {
	Service string          `json:"service"`
	Params  api.CoderParams `json:"params"`
}

// CoderEngine is set during initialization and used by CoderWorkflow.
var CoderEngine *coder.Engine

// CoderWorkflow is a DBOS durable workflow that runs a coder job.
func CoderWorkflow(ctx dbos.DBOSContext, input CoderWorkflowInput) (string, error) {
	wfID, err := dbos.GetWorkflowID(ctx)
	if err != nil {
		return "", fmt.Errorf("get workflow ID: %w", err)
	}

	result, err := dbos.RunAsStep(ctx, func(stepCtx context.Context) (string, error) {
		p := coder.Params{
			ProjectDir:   input.Params.ProjectDir,
			Prompt:       input.Params.Prompt,
			SystemPrompt: derefStr(input.Params.SystemPrompt),
			Model:        derefStr(input.Params.Model),
			ID:           wfID,
			EnvVars:      derefMap(input.Params.EnvVars),
		}
		report, err := CoderEngine.Run(stepCtx, p)
		if err != nil {
			return "", err
		}
		return report.Summary, nil
	}, dbos.WithStepName("run_coder"))

	return result, err
}

// DBOSJobManager implements api.JobManager using DBOS durable workflows.
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
	default:
		job.Status = api.Running
	}

	return job, nil
}

func newUUID() uuid.UUID {
	id, err := uuid.NewRandom()
	if err != nil {
		panic(fmt.Sprintf("failed to generate UUID: %v", err))
	}
	return id
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefMap(m *map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	return *m
}

// Verify DBOSJobManager implements api.JobManager at compile time.
var _ api.JobManager = (*DBOSJobManager)(nil)
