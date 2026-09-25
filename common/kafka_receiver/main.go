package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	observabilityv1 "OpenCNC/common/structures/logging"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Config holds command-line and environment settings.
type Config struct {
	Brokers             []string
	Topics              []string
	GroupID             string
	FromBeginning       bool
	OutputDir           string
	RawCSVFile          string
	SummaryCSVFile      string
	FlushInterval       time.Duration
	JSONTopic           string
	MonitoringJSONTopic string
}

// LatencySummary aggregates individual subsystem latencies for a given request.
type LatencySummary struct {
	FirstSeenAt      time.Time
	LastSeenAt       time.Time
	CorrelationID    string
	StreamID         string
	RequestID        string
	Interface        string // http or grpc
	TotalUNIMs       float64
	TSNCalcMs        float64
	ConfigAppMs      float64
	ConfigPrepMs     float64
	ConfigCommitMs   float64
	HasTotalUNI      bool
	HasTSNCalc       bool
	HasConfigApp     bool
	HasConfigPrep    bool
	HasConfigCommit  bool
	WrittenToSummary bool
	flushTimer       *time.Timer
}

type LatencySummaryJSON struct {
	TimestampUTC          string  `json:"timestamp"`
	CorrelationID         string  `json:"correlation_id"`
	StreamID              string  `json:"stream_id,omitempty"`
	RequestID             string  `json:"request_id,omitempty"`
	Interface             string  `json:"interface"`
	TotalUNILatencyMs     float64 `json:"total_uni_latency_ms"`
	TSNCalcLatencyMs      float64 `json:"tsn_calc_latency_ms"`
	ConfigAppLatencyMs    float64 `json:"config_app_latency_ms"`
	ConfigPrepLatencyMs   float64 `json:"config_prep_latency_ms"`
	ConfigCommitLatencyMs float64 `json:"config_commit_latency_ms"`
	MainServiceOverheadMs float64 `json:"main_service_overhead_ms"`
}

type MetricSample struct {
	Timestamp time.Time
	Value     float64
	Used      bool
}

type Receiver struct {
	cfg              Config
	rawFile          *os.File
	rawWriter        *csv.Writer
	summaryFile      *os.File
	summaryWriter    *csv.Writer
	monitoringWriter messageWriter
	jsonWriter       *kafka.Writer
	logWriter        *kafka.Writer

	mu        sync.Mutex
	summaries map[string]*LatencySummary

	// Track recent TotalUNI summary awaiting late-arriving subsystem metrics
	recentTotalUNI     *LatencySummary
	recentTSNCalc      *MetricSample
	recentConfigApp    *MetricSample
	recentConfigPrep   *MetricSample
	recentConfigCommit *MetricSample
}

func main() {
	var (
		brokersFlag       = flag.String("brokers", getEnv("OBS_BROKERS", "localhost:9092"), "Comma-separated Kafka broker addresses")
		topicsFlag        = flag.String("topics", getEnv("KAFKA_TOPICS", "opencnc.metrics,opencnc.events,opencnc.logs"), "Comma-separated Kafka topics to consume")
		groupFlag         = flag.String("group", getEnv("KAFKA_GROUP", ""), "Kafka consumer group ID (leave empty for fresh ephemeral consumer)")
		fromBeginningFlag = flag.Bool("from-beginning", false, "Consume from the beginning of topic (default false: only current run)")
		outputDirFlag     = flag.String("output-dir", getEnv("OUTPUT_DIR", "."), "Directory where CSV files will be stored")
		rawCSVFlag        = flag.String("raw-csv", "metrics_raw.csv", "Filename for raw metric records CSV")
		summaryCSVFlag    = flag.String("summary-csv", "stream_latencies_summary.csv", "Filename for aggregated stream latency summary CSV")
		jsonTopicFlag     = flag.String("json-topic", getEnv("KAFKA_JSON_TOPIC", "opencnc.latencies.json"), "Kafka topic to publish correlated latency summaries in JSON")
		flushSecFlag      = flag.Int("flush-seconds", 1, "Interval in seconds to flush CSV writers to disk")
	)
	monitoringTopicFlag := flag.String("monitoring-json-topic", getEnv("KAFKA_MONITORING_JSON_TOPIC", "opencnc.monitoring.json"), "JSON output topic for device metrics (empty disables)")
	flag.Parse()

	brokers := splitAndTrim(*brokersFlag)
	topics := splitAndTrim(*topicsFlag)

	cfg := Config{
		Brokers:             brokers,
		Topics:              topics,
		GroupID:             strings.TrimSpace(*groupFlag),
		FromBeginning:       *fromBeginningFlag,
		OutputDir:           *outputDirFlag,
		RawCSVFile:          *rawCSVFlag,
		SummaryCSVFile:      *summaryCSVFlag,
		FlushInterval:       time.Duration(*flushSecFlag) * time.Second,
		JSONTopic:           strings.TrimSpace(*jsonTopicFlag),
		MonitoringJSONTopic: strings.TrimSpace(*monitoringTopicFlag),
	}

	receiver, err := NewReceiver(cfg)
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize receiver: %v", err)
	}
	defer receiver.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("==================================================================")
	log.Printf(" OpenCNC Kafka Latency Receiver")
	log.Printf("==================================================================")
	log.Printf(" Brokers        : %s", strings.Join(cfg.Brokers, ", "))
	log.Printf(" Topics         : %s", strings.Join(cfg.Topics, ", "))
	log.Printf(" Consumer Group : %s", cfg.GroupID)
	log.Printf(" Output Dir     : %s", cfg.OutputDir)
	log.Printf(" Raw CSV        : %s", filepath.Join(cfg.OutputDir, cfg.RawCSVFile))
	log.Printf(" Summary CSV    : %s", filepath.Join(cfg.OutputDir, cfg.SummaryCSVFile))
	log.Printf("==================================================================")

	if err := receiver.Start(ctx); err != nil && err != context.Canceled {
		log.Printf("[ERROR] Receiver terminated with error: %v", err)
	}

	log.Println("[INFO] Shutting down Kafka latency receiver gracefully...")
}

func NewReceiver(cfg Config) (*Receiver, error) {
	for _, topic := range cfg.Topics {
		if cfg.MonitoringJSONTopic != "" && topic == cfg.MonitoringJSONTopic {
			return nil, fmt.Errorf("monitoring output topic must not be an input topic")
		}
	}
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory %s: %w", cfg.OutputDir, err)
	}

	rawPath := filepath.Join(cfg.OutputDir, cfg.RawCSVFile)
	rawFileExisted := fileExists(rawPath)
	rawFile, err := os.OpenFile(rawPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open raw CSV file: %w", err)
	}
	rawWriter := csv.NewWriter(rawFile)

	if !rawFileExisted {
		headers := []string{
			"timestamp_utc",
			"service",
			"event_kind",
			"metric_name",
			"value",
			"unit",
			"correlation_id",
			"event_id",
			"attributes_json",
			"details",
		}
		if err := rawWriter.Write(headers); err != nil {
			rawFile.Close()
			return nil, fmt.Errorf("failed to write raw CSV headers: %w", err)
		}
		rawWriter.Flush()
	}

	summaryPath := filepath.Join(cfg.OutputDir, cfg.SummaryCSVFile)
	summaryFileExisted := fileExists(summaryPath)
	summaryFile, err := os.OpenFile(summaryPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		rawFile.Close()
		return nil, fmt.Errorf("failed to open summary CSV file: %w", err)
	}
	summaryWriter := csv.NewWriter(summaryFile)

	if !summaryFileExisted {
		headers := []string{
			"timestamp_utc",
			"correlation_id",
			"stream_id",
			"request_id",
			"interface",
			"total_uni_latency_ms",
			"tsn_calc_latency_ms",
			"config_app_latency_ms",
			"config_prep_latency_ms",
			"config_commit_latency_ms",
			"main_service_overhead_ms",
		}
		if err := summaryWriter.Write(headers); err != nil {
			rawFile.Close()
			summaryFile.Close()
			return nil, fmt.Errorf("failed to write summary CSV headers: %w", err)
		}
		summaryWriter.Flush()
	}

	var jsonWriter *kafka.Writer
	if cfg.JSONTopic != "" && len(cfg.Brokers) > 0 {
		jsonWriter = &kafka.Writer{
			Addr:         kafka.TCP(cfg.Brokers...),
			Topic:        cfg.JSONTopic,
			Balancer:     &kafka.Hash{},
			BatchTimeout: 10 * time.Millisecond,
			RequiredAcks: kafka.RequireOne,
		}
	}

	var logWriter *kafka.Writer
	if len(cfg.Brokers) > 0 {
		logWriter = &kafka.Writer{
			Addr:         kafka.TCP(cfg.Brokers...),
			Balancer:     &kafka.Hash{},
			BatchTimeout: 10 * time.Millisecond,
			RequiredAcks: kafka.RequireOne,
		}
	}

	var monitoringWriter messageWriter
	if cfg.MonitoringJSONTopic != "" && len(cfg.Brokers) > 0 {
		monitoringWriter = &kafka.Writer{Addr: kafka.TCP(cfg.Brokers...), Topic: cfg.MonitoringJSONTopic, Balancer: &kafka.Hash{}, RequiredAcks: kafka.RequireAll, BatchTimeout: 10 * time.Millisecond, WriteTimeout: 10 * time.Second}
	}
	return &Receiver{
		monitoringWriter: monitoringWriter,
		cfg:              cfg,
		rawFile:          rawFile,
		rawWriter:        rawWriter,
		summaryFile:      summaryFile,
		summaryWriter:    summaryWriter,
		jsonWriter:       jsonWriter,
		logWriter:        logWriter,
		summaries:        make(map[string]*LatencySummary),
	}, nil
}

func (r *Receiver) Start(ctx context.Context) error {
	var wg sync.WaitGroup

	// Periodic flusher for CSV buffers
	wg.Add(1)
	go func() { defer wg.Done(); r.periodicFlusher(ctx) }()

	// Launch consumer goroutine for each topic
	for _, topic := range r.cfg.Topics {
		wg.Add(1)
		go func(t string) {
			defer wg.Done()
			r.consumeTopic(ctx, t)
		}(topic)
	}

	<-ctx.Done()
	wg.Wait()
	return ctx.Err()
}

func (r *Receiver) consumeTopic(ctx context.Context, topic string) {
	readerConfig := kafka.ReaderConfig{
		Brokers:  r.cfg.Brokers,
		Topic:    topic,
		MinBytes: 10e1, // 100B
		MaxBytes: 10e6, // 10MB
	}

	if r.cfg.GroupID != "" {
		readerConfig.GroupID = r.cfg.GroupID + "." + topic
	} else {
		// Use a unique ephemeral group ID for this run to avoid replaying previous runs
		readerConfig.GroupID = fmt.Sprintf("opencnc-run-%d", time.Now().UnixNano())
	}

	if r.cfg.FromBeginning {
		readerConfig.StartOffset = kafka.FirstOffset
	} else {
		readerConfig.StartOffset = kafka.LastOffset
	}

	reader := kafka.NewReader(readerConfig)
	defer reader.Close()

	log.Printf("[INFO] Consumer started for topic: %s", topic)

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[WARN] Error reading from topic %s: %v. Retrying in 2s...", topic, err)
			select {
			case <-time.After(2 * time.Second):
				continue
			case <-ctx.Done():
				return
			}
		}

		if err := r.forwardMonitoring(ctx, msg); err != nil {
			return
		}
		r.processMessage(msg)
		for {
			if err := reader.CommitMessages(ctx, msg); err == nil {
				break
			} else {
				log.Printf("[WARN] Commit failed for %s/%d/%d: %v", msg.Topic, msg.Partition, msg.Offset, err)
			}
			if !waitRetry(ctx) {
				return
			}
		}
	}
}

func (r *Receiver) processMessage(msg kafka.Message) {
	var event observabilityv1.EventEnvelope

	// First try unmarshaling as Protobuf binary
	err := proto.Unmarshal(msg.Value, &event)
	if err != nil {
		// Fallback to Protobuf JSON unmarshaling
		err = protojson.Unmarshal(msg.Value, &event)
	}

	if err == nil && (event.GetKind() != observabilityv1.EventKind_EVENT_KIND_UNSPECIFIED || event.Payload != nil) {
		r.processEnvelope(msg, &event)
		return
	}

	// Fallback: Message might be a generic JSON document or raw text
	r.processGenericPayload(msg)
}

func (r *Receiver) processEnvelope(msg kafka.Message, event *observabilityv1.EventEnvelope) {
	occurredAt := time.Now().UTC()
	if event.GetOccurredAt() != nil {
		occurredAt = event.GetOccurredAt().AsTime().UTC()
	}

	service := event.GetSource().GetService()
	correlationID := event.GetTrace().GetCorrelationId()
	eventID := event.GetEventId()
	eventKind := event.GetKind().String()

	switch payload := event.Payload.(type) {
	case *observabilityv1.EventEnvelope_Metric:
		metric := payload.Metric
		if metric == nil {
			return
		}

		attrJSON, _ := json.Marshal(metric.GetAttributes())

		log.Printf(
			"[METRIC] Topic: %-15s | Service: %-14s | Metric: %-34s | Value: %8.2f %-4s | CorrID: %s",
			msg.Topic,
			service,
			metric.GetName(),
			metric.GetValue(),
			metric.GetUnit(),
			truncate(correlationID, 12),
		)

		record := []string{
			occurredAt.Format(time.RFC3339Nano),
			service,
			eventKind,
			metric.GetName(),
			strconv.FormatFloat(metric.GetValue(), 'f', 4, 64),
			metric.GetUnit(),
			correlationID,
			eventID,
			string(attrJSON),
			"",
		}

		r.mu.Lock()
		_ = r.rawWriter.Write(record)
		r.rawWriter.Flush()
		if normalizeServiceName(service) != "monitor-service" {
			r.updateSummaryLocked(occurredAt, service, correlationID, metric)
		}
		r.mu.Unlock()

	case *observabilityv1.EventEnvelope_Domain:
		domain := payload.Domain
		if domain == nil {
			return
		}

		details := fmt.Sprintf("%s:%s result=%s subject=%s:%s summary=%s",
			domain.GetDomain(), domain.GetAction(), domain.GetResult().String(),
			domain.GetSubjectType(), domain.GetSubjectId(), domain.GetSummary(),
		)

		log.Printf("[DOMAIN EVENT] Service: %-14s | %s", service, details)

		record := []string{
			occurredAt.Format(time.RFC3339Nano),
			service,
			eventKind,
			domain.GetDomain() + "." + domain.GetAction(),
			"",
			"",
			correlationID,
			eventID,
			"",
			details,
		}

		r.mu.Lock()
		_ = r.rawWriter.Write(record)
		r.rawWriter.Flush()
		r.mu.Unlock()

	case *observabilityv1.EventEnvelope_Log:
		logEvent := payload.Log
		if logEvent == nil {
			return
		}

		fieldsJSON, _ := json.Marshal(logEvent.GetFields())
		log.Printf("[LOG EVENT] Service: %-14s | Severity: %-5s | Msg: %s", service, event.GetSeverity().String(), logEvent.GetMessage())

		record := []string{
			occurredAt.Format(time.RFC3339Nano),
			service,
			eventKind,
			"log." + event.GetSeverity().String(),
			"",
			"",
			correlationID,
			eventID,
			string(fieldsJSON),
			logEvent.GetMessage(),
		}

		r.mu.Lock()
		_ = r.rawWriter.Write(record)
		r.rawWriter.Flush()
		r.mu.Unlock()

		r.forwardCleanLog(occurredAt, service, event.GetSeverity().String(), logEvent.GetMessage(), correlationID)

	case *observabilityv1.EventEnvelope_Audit:
		audit := payload.Audit
		if audit == nil {
			return
		}

		details := fmt.Sprintf("action=%s actor=%s target=%s:%s result=%s reason=%s",
			audit.GetAction(), audit.GetActor(), audit.GetTargetType(), audit.GetTargetId(), audit.GetResult().String(), audit.GetReason())
		log.Printf("[AUDIT EVENT] Service: %-14s | %s", service, details)

		record := []string{
			occurredAt.Format(time.RFC3339Nano),
			service,
			eventKind,
			"audit." + audit.GetAction(),
			"",
			"",
			correlationID,
			eventID,
			"",
			details,
		}

		r.mu.Lock()
		_ = r.rawWriter.Write(record)
		r.rawWriter.Flush()
		r.mu.Unlock()

		r.forwardCleanLog(occurredAt, service, event.GetSeverity().String(), details, correlationID)

	case *observabilityv1.EventEnvelope_Health:
		health := payload.Health
		if health == nil {
			return
		}

		details := fmt.Sprintf("check=%s status=%s message=%s", health.GetCheck(), health.GetStatus().String(), health.GetMessage())
		log.Printf("[HEALTH EVENT] Service: %-14s | %s", service, details)

		record := []string{
			occurredAt.Format(time.RFC3339Nano),
			service,
			eventKind,
			"health." + health.GetCheck(),
			"",
			"",
			correlationID,
			eventID,
			"",
			details,
		}

		r.mu.Lock()
		_ = r.rawWriter.Write(record)
		r.rawWriter.Flush()
		r.mu.Unlock()

		r.forwardCleanLog(occurredAt, service, event.GetSeverity().String(), details, correlationID)
	}
}

// processGenericPayload handles non-protobuf JSON or raw payloads and writes them to the CSV.
func (r *Receiver) processGenericPayload(msg kafka.Message) {
	occurredAt := msg.Time.UTC()
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	var jsonMap map[string]interface{}
	if err := json.Unmarshal(msg.Value, &jsonMap); err == nil {
		service := "unknown"
		if s, ok := jsonMap["service"].(string); ok {
			service = s
		}
		metricName := msg.Topic
		if m, ok := jsonMap["metric"].(string); ok {
			metricName = m
		} else if m, ok := jsonMap["metric_name"].(string); ok {
			metricName = m
		}
		valStr := ""
		if v, ok := jsonMap["value"].(float64); ok {
			valStr = strconv.FormatFloat(v, 'f', 4, 64)
		}
		unitStr := ""
		if u, ok := jsonMap["unit"].(string); ok {
			unitStr = u
		}
		corrID := string(msg.Key)
		if c, ok := jsonMap["correlation_id"].(string); ok {
			corrID = c
		}

		record := []string{
			occurredAt.Format(time.RFC3339Nano),
			service,
			"JSON",
			metricName,
			valStr,
			unitStr,
			corrID,
			"",
			string(msg.Value),
			"",
		}

		r.mu.Lock()
		_ = r.rawWriter.Write(record)
		r.rawWriter.Flush()
		r.mu.Unlock()

		if msg.Topic == "opencnc.logs" {
			if msgText, ok := jsonMap["message"].(string); ok {
				r.forwardCleanLog(occurredAt, service, "INFO", msgText, corrID)
			}
		}
		return
	}

	// Plain text fallback
	record := []string{
		occurredAt.Format(time.RFC3339Nano),
		"raw",
		"TEXT",
		msg.Topic,
		"",
		"",
		string(msg.Key),
		"",
		"",
		string(msg.Value),
	}

	r.mu.Lock()
	_ = r.rawWriter.Write(record)
	r.rawWriter.Flush()
	r.mu.Unlock()

	if msg.Topic == "opencnc.logs" {
		r.forwardCleanLog(occurredAt, "unknown", "INFO", string(msg.Value), string(msg.Key))
	}
}

func (r *Receiver) updateSummaryLocked(
	occurredAt time.Time,
	service string,
	correlationID string,
	metric *observabilityv1.MetricEvent,
) {
	if correlationID == "" {
		correlationID = "unknown-" + occurredAt.Format("150405")
	}

	summary, ok := r.summaries[correlationID]
	if !ok {
		summary = &LatencySummary{
			FirstSeenAt:   occurredAt,
			CorrelationID: correlationID,
		}
		r.summaries[correlationID] = summary
	}

	summary.LastSeenAt = occurredAt

	if metric.Attributes != nil {
		if sid, ok := metric.Attributes["stream_id"]; ok && sid != "" {
			summary.StreamID = sid
		}
		if rid, ok := metric.Attributes["request_id"]; ok && rid != "" {
			summary.RequestID = rid
		}
		if iface, ok := metric.Attributes["interface"]; ok && iface != "" {
			summary.Interface = iface
		}
	}

	switch metric.GetName() {
	case "uni_request_response_duration":
		summary.TotalUNIMs = metric.GetValue()
		summary.HasTotalUNI = true
		if summary.Interface == "" && metric.Attributes != nil {
			summary.Interface = metric.Attributes["interface"]
		}

		// Correlate with recent subsystem metrics that completed within this request window
		windowDuration := time.Duration(summary.TotalUNIMs*1.2)*time.Millisecond + 15*time.Second
		windowStart := occurredAt.Add(-windowDuration)

		if !summary.HasTSNCalc && r.recentTSNCalc != nil && !r.recentTSNCalc.Used && r.recentTSNCalc.Timestamp.After(windowStart) {
			summary.TSNCalcMs = r.recentTSNCalc.Value
			summary.HasTSNCalc = true
			r.recentTSNCalc.Used = true
		}

		if !summary.HasConfigApp && r.recentConfigApp != nil && !r.recentConfigApp.Used && r.recentConfigApp.Timestamp.After(windowStart) {
			summary.ConfigAppMs = r.recentConfigApp.Value
			summary.HasConfigApp = true
			r.recentConfigApp.Used = true
		}

		if !summary.HasConfigPrep && r.recentConfigPrep != nil && !r.recentConfigPrep.Used && r.recentConfigPrep.Timestamp.After(windowStart) {
			summary.ConfigPrepMs = r.recentConfigPrep.Value
			summary.HasConfigPrep = true
			r.recentConfigPrep.Used = true
		}

		if !summary.HasConfigCommit && r.recentConfigCommit != nil && !r.recentConfigCommit.Used && r.recentConfigCommit.Timestamp.After(windowStart) {
			summary.ConfigCommitMs = r.recentConfigCommit.Value
			summary.HasConfigCommit = true
			r.recentConfigCommit.Used = true
		}

		r.recentTotalUNI = summary

		// If all primary subsystems are correlated, write immediately;
		// otherwise give late-arriving subsystem metrics up to 800ms to arrive.
		if summary.HasTSNCalc && summary.HasConfigApp {
			r.writeSummaryRowLocked(summary)
		} else {
			if summary.flushTimer != nil {
				summary.flushTimer.Stop()
			}
			summary.flushTimer = time.AfterFunc(800*time.Millisecond, func() {
				r.mu.Lock()
				defer r.mu.Unlock()
				if !summary.WrittenToSummary {
					r.writeSummaryRowLocked(summary)
				}
			})
		}

	case "configuration_calculation_duration":
		summary.TSNCalcMs = metric.GetValue()
		summary.HasTSNCalc = true
		r.recentTSNCalc = &MetricSample{Timestamp: occurredAt, Value: metric.GetValue()}
		// Symmetrical correlation: if TotalUNI is waiting, update it
		if r.recentTotalUNI != nil && !r.recentTotalUNI.HasTSNCalc && occurredAt.Sub(r.recentTotalUNI.LastSeenAt) < 15*time.Second {
			r.recentTotalUNI.TSNCalcMs = metric.GetValue()
			r.recentTotalUNI.HasTSNCalc = true
			r.recentTSNCalc.Used = true
			if r.recentTotalUNI.HasConfigApp && !r.recentTotalUNI.WrittenToSummary {
				if r.recentTotalUNI.flushTimer != nil {
					r.recentTotalUNI.flushTimer.Stop()
				}
				r.writeSummaryRowLocked(r.recentTotalUNI)
			}
		}

	case "configuration_application_duration":
		summary.ConfigAppMs = metric.GetValue()
		summary.HasConfigApp = true
		r.recentConfigApp = &MetricSample{Timestamp: occurredAt, Value: metric.GetValue()}
		// Symmetrical correlation: if TotalUNI is waiting, update it
		if r.recentTotalUNI != nil && !r.recentTotalUNI.HasConfigApp && occurredAt.Sub(r.recentTotalUNI.LastSeenAt) < 15*time.Second {
			r.recentTotalUNI.ConfigAppMs = metric.GetValue()
			r.recentTotalUNI.HasConfigApp = true
			r.recentConfigApp.Used = true
			if r.recentTotalUNI.HasTSNCalc && !r.recentTotalUNI.WrittenToSummary {
				if r.recentTotalUNI.flushTimer != nil {
					r.recentTotalUNI.flushTimer.Stop()
				}
				r.writeSummaryRowLocked(r.recentTotalUNI)
			}
		}

	case "configuration_preparation_duration":
		summary.ConfigPrepMs = metric.GetValue()
		summary.HasConfigPrep = true
		r.recentConfigPrep = &MetricSample{Timestamp: occurredAt, Value: metric.GetValue()}
		if r.recentTotalUNI != nil && !r.recentTotalUNI.HasConfigPrep && occurredAt.Sub(r.recentTotalUNI.LastSeenAt) < 15*time.Second {
			r.recentTotalUNI.ConfigPrepMs = metric.GetValue()
			r.recentTotalUNI.HasConfigPrep = true
			r.recentConfigPrep.Used = true
		}

	case "configuration_commit_duration":
		summary.ConfigCommitMs = metric.GetValue()
		summary.HasConfigCommit = true
		r.recentConfigCommit = &MetricSample{Timestamp: occurredAt, Value: metric.GetValue()}
		if r.recentTotalUNI != nil && !r.recentTotalUNI.HasConfigCommit && occurredAt.Sub(r.recentTotalUNI.LastSeenAt) < 15*time.Second {
			r.recentTotalUNI.ConfigCommitMs = metric.GetValue()
			r.recentTotalUNI.HasConfigCommit = true
			r.recentConfigCommit.Used = true
		}
	}
}

func (r *Receiver) writeSummaryRowLocked(s *LatencySummary) {
	if s.WrittenToSummary {
		return
	}
	s.WrittenToSummary = true

	mainOverhead := 0.0
	if s.HasTotalUNI && (s.HasTSNCalc || s.HasConfigApp) {
		mainOverhead = s.TotalUNIMs - (s.TSNCalcMs + s.ConfigAppMs)
		if mainOverhead < 0 {
			mainOverhead = 0
		}
	}

	record := []string{
		s.LastSeenAt.Format(time.RFC3339Nano),
		s.CorrelationID,
		s.StreamID,
		s.RequestID,
		s.Interface,
		fmtFloat(s.TotalUNIMs, s.HasTotalUNI),
		fmtFloat(s.TSNCalcMs, s.HasTSNCalc),
		fmtFloat(s.ConfigAppMs, s.HasConfigApp),
		fmtFloat(s.ConfigPrepMs, s.HasConfigPrep),
		fmtFloat(s.ConfigCommitMs, s.HasConfigCommit),
		fmt.Sprintf("%.2f", mainOverhead),
	}

	_ = r.summaryWriter.Write(record)
	r.summaryWriter.Flush()

	log.Printf("==================================================================")
	log.Printf(" [SUMMARY RECORDED] Stream Latency Breakdown")
	log.Printf(" Correlation ID  : %s", s.CorrelationID)
	log.Printf(" Total UNI Time  : %s ms (%s)", fmtFloat(s.TotalUNIMs, s.HasTotalUNI), s.Interface)
	log.Printf(" TSN Service     : %s ms", fmtFloat(s.TSNCalcMs, s.HasTSNCalc))
	log.Printf(" Config Service  : %s ms (prep=%s ms, commit=%s ms)",
		fmtFloat(s.ConfigAppMs, s.HasConfigApp),
		fmtFloat(s.ConfigPrepMs, s.HasConfigPrep),
		fmtFloat(s.ConfigCommitMs, s.HasConfigCommit),
	)
	log.Printf(" Main Overhead   : %.2f ms", mainOverhead)
	log.Printf("==================================================================")

	// Publish JSON event to Kafka topic for real-time Grafana ingestion
	if r.jsonWriter != nil {
		jsonPayload := LatencySummaryJSON{
			TimestampUTC:          s.LastSeenAt.Format(time.RFC3339Nano),
			CorrelationID:         s.CorrelationID,
			StreamID:              s.StreamID,
			RequestID:             s.RequestID,
			Interface:             s.Interface,
			TotalUNILatencyMs:     s.TotalUNIMs,
			TSNCalcLatencyMs:      s.TSNCalcMs,
			ConfigAppLatencyMs:    s.ConfigAppMs,
			ConfigPrepLatencyMs:   s.ConfigPrepMs,
			ConfigCommitLatencyMs: s.ConfigCommitMs,
			MainServiceOverheadMs: mainOverhead,
		}

		if data, err := json.Marshal(jsonPayload); err == nil {
			msg := kafka.Message{
				Key:   []byte(s.CorrelationID),
				Value: data,
				Time:  s.LastSeenAt,
			}
			go func(m kafka.Message) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := r.jsonWriter.WriteMessages(ctx, m); err != nil {
					log.Printf("[WARN] Failed publishing JSON latency to topic %s: %v", r.cfg.JSONTopic, err)
				} else {
					log.Printf("[INFO] Published JSON latency summary to Kafka topic: %s", r.cfg.JSONTopic)
				}
			}(msg)
		} else {
			log.Printf("[WARN] Failed marshaling JSON latency summary: %v", err)
		}
	}
}

func (r *Receiver) periodicFlusher(ctx context.Context) {
	ticker := time.NewTicker(r.cfg.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.mu.Lock()
			r.rawWriter.Flush()
			r.summaryWriter.Flush()
			r.mu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

func (r *Receiver) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.rawWriter != nil {
		r.rawWriter.Flush()
	}
	if r.summaryWriter != nil {
		r.summaryWriter.Flush()
	}
	if r.monitoringWriter != nil {
		_ = r.monitoringWriter.Close()
	}
	if r.jsonWriter != nil {
		_ = r.jsonWriter.Close()
	}
	if r.logWriter != nil {
		_ = r.logWriter.Close()
	}
	if r.rawFile != nil {
		_ = r.rawFile.Sync()
		_ = r.rawFile.Close()
	}
	if r.summaryFile != nil {
		_ = r.summaryFile.Sync()
		_ = r.summaryFile.Close()
	}
}

func normalizeServiceName(service string) string {
	s := strings.ToLower(strings.TrimSpace(service))
	if strings.Contains(s, "main") {
		return "main-service"
	}
	if strings.Contains(s, "tsn") {
		return "tsn-service"
	}
	if strings.Contains(s, "config") {
		return "config-service"
	}
	if strings.Contains(s, "gui") {
		return "gui-service"
	}
	if strings.Contains(s, "monitor") {
		return "monitor-service"
	}
	if s == "" {
		return "unknown"
	}
	return s
}

func (r *Receiver) forwardCleanLog(occurredAt time.Time, rawService string, severity string, msgText string, corrID string) {
	if r.logWriter == nil || strings.TrimSpace(msgText) == "" {
		return
	}
	service := normalizeServiceName(rawService)
	if severity == "" {
		severity = "INFO"
	}
	severity = strings.TrimPrefix(severity, "SEVERITY_")

	// Standard log line format that Grafana parses cleanly without any unknown symbols:
	// Example: 2026-09-24 23:55:00.123 [INFO] [main-service] Tentative configuration retrieved...
	timeStr := occurredAt.UTC().Format("2006-01-02 15:04:05.000")
	cleanLine := fmt.Sprintf("%s [%s] [%s] %s", timeStr, severity, service, strings.TrimSpace(msgText))
	if corrID != "" {
		cleanLine += fmt.Sprintf(" (corr_id=%s)", corrID)
	}

	valBytes := []byte(cleanLine)
	keyBytes := []byte(corrID)
	if len(keyBytes) == 0 {
		keyBytes = []byte(service)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 1. Publish to the service-specific topic (e.g. opencnc.logs.config-service)
	svcTopic := fmt.Sprintf("opencnc.logs.%s", service)
	_ = r.logWriter.WriteMessages(ctx, kafka.Message{
		Topic: svcTopic,
		Key:   keyBytes,
		Value: valBytes,
		Time:  occurredAt,
	})

	// 2. Publish to the unified clean topic
	_ = r.logWriter.WriteMessages(ctx, kafka.Message{
		Topic: "opencnc.logs.clean",
		Key:   keyBytes,
		Value: valBytes,
		Time:  occurredAt,
	})
}

func fmtFloat(val float64, present bool) string {
	if !present {
		return "N/A"
	}
	return fmt.Sprintf("%.2f", val)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir() && info.Size() > 0
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			res = append(res, trimmed)
		}
	}
	return res
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
