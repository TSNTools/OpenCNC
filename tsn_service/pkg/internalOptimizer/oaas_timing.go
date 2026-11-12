package internalOptimizer

import (
	"context"
	"time"

	"OpenCNC/common/observability"
	observabilityv1 "OpenCNC/common/structures/logging"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// oaasTimingRecorder stores elapsed, monotonic-clock durations until the
// in-scope optimization call has finished. Kafka publishing is not part of
// the measured operations.
type oaasTimingRecorder struct {
	obs     *observability.Client
	runID   string
	jobID   string
	samples []oaasTimingSample
}

type oaasTimingSample struct {
	name     string
	end      time.Time
	duration time.Duration
	status   string
}

func newOAASTimingRecorder(obs *observability.Client) *oaasTimingRecorder {
	if obs == nil {
		return nil
	}
	return &oaasTimingRecorder{obs: obs, runID: uuid.NewString()}
}

func (r *oaasTimingRecorder) record(name string, started time.Time, success bool) {
	if r == nil {
		return
	}
	ended := time.Now()
	status := "success"
	if !success {
		status = "failed"
	}
	r.samples = append(r.samples, oaasTimingSample{
		name: name, end: ended, duration: ended.Sub(started), status: status,
	})
}

// publish schedules one bounded, best-effort delivery attempt for all stages.
// The snapshot avoids races with the finished request and does not hold up the
// FPM return while Kafka is unavailable. Delivery must still be verified.
func (r *oaasTimingRecorder) publish() <-chan struct{} {
	done := make(chan struct{})
	if r == nil || r.obs == nil || len(r.samples) == 0 {
		close(done)
		return done
	}
	samples := append([]oaasTimingSample(nil), r.samples...)
	obs, runID, jobID := r.obs, r.runID, r.jobID
	go func() {
		defer close(done)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, sample := range samples {
			if ctx.Err() != nil {
				return
			}
			attributes := map[string]string{
				"run_id":       runID,
				"stage_status": sample.status,
			}
			if sample.name == "oaas_request_build_ms" {
				attributes["build_status"] = sample.status
			}
			if jobID != "" {
				attributes["job_id"] = jobID
			}
			_ = obs.MetricAt(
				ctx,
				timestamppb.New(sample.end),
				observabilityv1.Severity_SEVERITY_INFO,
				sample.name,
				observabilityv1.MetricType_METRIC_TYPE_GAUGE,
				float64(sample.duration)/float64(time.Millisecond),
				"ms",
				attributes,
			)
		}
	}()
	return done
}
