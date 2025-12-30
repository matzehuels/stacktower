package api

import (
	"time"

	"github.com/matzehuels/stacktower/pkg/infra/queue"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// JobRequest contains the common fields needed to create a job.
type JobRequest struct {
	UserID    string
	TraceID   string
	Options   pipeline.Options
	GraphID   string // For layout jobs
	GraphData []byte // For layout jobs with inline graph
}

// newJob creates a queue.Job from a JobRequest.
// This is the single point of job creation to ensure consistency.
func newJob(jobType queue.Type, req JobRequest) (*queue.Job, error) {
	payload := &pipeline.JobPayload{
		Options:   req.Options,
		TraceID:   req.TraceID,
		GraphID:   req.GraphID,
		GraphData: req.GraphData,
	}
	payload.Options.UserID = req.UserID

	payloadMap, err := payload.ToMap()
	if err != nil {
		return nil, err
	}

	return &queue.Job{
		ID:        generateJobID(),
		Type:      string(jobType),
		Payload:   payloadMap,
		Status:    queue.StatusPending,
		CreatedAt: time.Now(),
	}, nil
}

// newParseJob creates a parse job.
func newParseJob(req JobRequest) (*queue.Job, error) {
	return newJob(queue.TypeParse, req)
}

// newLayoutJob creates a layout job.
func newLayoutJob(req JobRequest) (*queue.Job, error) {
	return newJob(queue.TypeLayout, req)
}

// newRenderJob creates a render job.
func newRenderJob(req JobRequest) (*queue.Job, error) {
	return newJob(queue.TypeRender, req)
}
