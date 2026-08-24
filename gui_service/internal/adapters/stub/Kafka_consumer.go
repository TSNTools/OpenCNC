package stub

import (
	"context"
	"log"
	"time"

	observabilityv1 "OpenCNC_config_service/common/structures/logging"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

const (
	kafkaBroker   = "localhost:9092"
	metricsTopic  = "opencnc.metrics"
	consumerGroup = "opencnc-gui-consumer"

	kafkaRetryDelay = 2 * time.Second
)

type MetricsConsumer struct {
	reader *kafka.Reader
	state  *MonitoringState
}

func NewMetricsConsumer(state *MonitoringState) *MetricsConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaBroker},
		Topic:   metricsTopic,
		GroupID: consumerGroup,

		// If this consumer group has no committed offset yet,
		// start consuming from the beginning of the topic.
		StartOffset: kafka.FirstOffset,
	})

	return &MetricsConsumer{
		reader: reader,
		state:  state,
	}
}

func (c *MetricsConsumer) Run(ctx context.Context) error {
	log.Printf(
		"Starting GUI metrics consumer: topic=%s group=%s",
		metricsTopic,
		consumerGroup,
	)

	for {
		// Don't consume metrics when the GUI has not requested any.
		if !c.state.IsMonitoring() {
			select {
			case <-ctx.Done():
				log.Println("GUI metrics consumer stopped")
				return nil

			case <-time.After(200 * time.Millisecond):
				continue
			}
		}

		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			// Context was cancelled. This is a normal shutdown.
			if ctx.Err() != nil {
				log.Println("GUI metrics consumer stopped")
				return nil
			}

			// Kafka is unavailable or another read error occurred.
			log.Printf(
				"failed to read from Kafka: %v; retrying in %s",
				err,
				kafkaRetryDelay,
			)

			// Wait before trying again, but don't prevent shutdown.
			select {
			case <-time.After(kafkaRetryDelay):
				continue

			case <-ctx.Done():
				log.Println("GUI metrics consumer stopped")
				return nil
			}
		}

		var event observabilityv1.EventEnvelope

		if err := proto.Unmarshal(msg.Value, &event); err != nil {
			log.Printf(
				"failed to decode Kafka metric message: %v",
				err,
			)
			continue
		}

		c.handleMetric(&event)
	}
}

func (c *MetricsConsumer) Close() error {
	if c == nil || c.reader == nil {
		return nil
	}

	return c.reader.Close()
}

func (c *MetricsConsumer) handleMetric(event *observabilityv1.EventEnvelope) {
	if event == nil {
		return
	}

	if event.GetKind() != observabilityv1.EventKind_EVENT_KIND_METRIC {
		return
	}

	metricPayload := event.GetMetric()
	if metricPayload == nil {
		return
	}

	// Only keep metrics that the GUI currently requested.
	if !c.state.WantsMetric(metricPayload.GetName()) {
		return
	}

	var timestamp int64
	if event.GetOccurredAt() != nil {
		timestamp = event.GetOccurredAt().AsTime().UnixMilli()
	}

	c.state.SetMetric(MetricData{
		Name:      metricPayload.GetName(),
		Value:     metricPayload.GetValue(),
		Timestamp: timestamp,
	})

	// log.Printf(
	// 	"metric event=%s source=%s metric=%s value=%v",
	// 	event.GetEventId(),
	// 	event.GetSource().GetService(),
	// 	metricPayload.GetName(),
	// 	metricPayload.GetValue(),
	// )
	log.Printf(
		"metric=%s value=%v timestamp=%d",
		metricPayload.GetName(),
		metricPayload.GetValue(),
		timestamp,
	)
}
