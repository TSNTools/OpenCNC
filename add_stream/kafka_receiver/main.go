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
	Brokers       []string
	Topics        []string
	GroupID       string
	FromBeginning bool
	OutputDir     string
	RawCSVFile    string
	SummaryCSVFile string
	FlushInterval time.Duration
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
}

type MetricSample struct {
	Timestamp time.Time
	Value     float64
	Used      bool
}

type Receiver struct {
	cfg           Config
	rawFile       *os.File
	rawWriter     *csv.Writer
	summaryFile   *os.File
	summaryWriter *csv.Writer

	mu        sync.Mutex
	summaries map[string]*LatencySummary

	// Buffer for correlating subsystem metrics when CorrelationID is not shared across services
	recentTSNCalc      *MetricSample
	recentConfigApp    *MetricSample
	recentConfigPrep   *MetricSample
	recentConfigCommit *MetricSample
}

func main() {
	var (
		brokersFlag       = flag.String("brokers", getEnv("OBS_BROKERS", "localhost:9092"), "Comma-separated Kafka broker addresses")
		topicsFlag        = flag.String("topics", getEnv("KAFKA_TOPICS", "opencnc.metrics,opencnc.events"), "Comma-separated Kafka topics to consume")
		groupFlag         = flag.String("group", getEnv("KAFKA_GROUP", ""), "Kafka consumer group ID (leave empty for fresh ephemeral consumer)")
		fromBeginningFlag = flag.Bool("from-beginning", false, "Consume from the beginning of topic (default false: only current run)")
		outputDirFlag     = flag.String("output-dir", getEnv("OUTPUT_DIR", "."), "Directory where CSV files will be stored")
		rawCSVFlag        = flag.String("raw-csv", "metrics_raw.csv", "Filename for raw metric records CSV")
		summaryCSVFlag    = flag.String("summary-csv", "stream_latencies_summary.csv", "Filename for aggregated stream latency summary CSV")
		flushSecFlag      = flag.Int("flush-seconds", 2, "Interval in seconds to flush CSV writers to disk")
	)
	flag.Parse()

	brokers := splitAndTrim(*brokersFlag)
	topics := splitAndTrim(*topicsFlag)

	cfg := Config{
		Brokers:        brokers,
		Topics:         topics,
		GroupID:        strings.TrimSpace(*groupFlag),
		FromBeginning:  *fromBeginningFlag,
		OutputDir:      *outputDirFlag,
		RawCSVFile:     *rawCSVFlag,
		SummaryCSVFile: *summaryCSVFlag,
		FlushInterval:  time.Duration(*flushSecFlag) * time.Second,
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
	log.Printf(" Raw CSV        : %s", filepath.Join(cfg.OutputDir, cfg.RawCSVFile))
	log.Printf(" Summary CSV    : %s", filepath.Join(cfg.OutputDir, cfg.SummaryCSVFile))
	log.Printf("==================================================================")

	if err := receiver.Start(ctx); err != nil && err != context.Canceled {
		log.Printf("[ERROR] Receiver terminated with error: %v", err)
	}

	log.Println("[INFO] Shutting down Kafka latency receiver gracefully...")
}

func NewReceiver(cfg Config) (*Receiver, error) {
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

	return &Receiver{
		cfg:           cfg,
		rawFile:       rawFile,
		rawWriter:     rawWriter,
		summaryFile:   summaryFile,
		summaryWriter: summaryWriter,
		summaries:     make(map[string]*LatencySummary),
	}, nil
}

func (r *Receiver) Start(ctx context.Context) error {
	var wg sync.WaitGroup

	// Periodic flusher for CSV buffers
	go r.periodicFlusher(ctx)

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
		readerConfig.GroupID = r.cfg.GroupID
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
		msg, err := reader.ReadMessage(ctx)
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

		r.processMessage(msg)
	}
}

func (r *Receiver) processMessage(msg kafka.Message) {
	var event observabilityv1.EventEnvelope

	// First try unmarshaling as Protobuf binary
	err := proto.Unmarshal(msg.Value, &event)
	if err != nil {
		// Fallback to JSON unmarshaling
		err = protojson.Unmarshal(msg.Value, &event)
		if err != nil {
			log.Printf("[DEBUG] Non-protobuf message on topic %s (len=%d): %v", msg.Topic, len(msg.Value), err)
			return
		}
	}

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

		// Console feedback for key latency metrics
		log.Printf(
			"[METRIC] Topic: %-15s | Service: %-14s | Metric: %-34s | Value: %8.2f %-4s | CorrID: %s",
			msg.Topic,
			service,
			metric.GetName(),
			metric.GetValue(),
			metric.GetUnit(),
			truncate(correlationID, 12),
		)

		// Record in raw CSV
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
		r.updateSummaryLocked(occurredAt, service, correlationID, metric)
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
		r.mu.Unlock()
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

		// Since OpenCNC generates a distinct random CorrelationID for each metric event,
		// correlate with the most recent subsystem metrics that completed within this request window:
		windowDuration := time.Duration(summary.TotalUNIMs*1.2)*time.Millisecond + 10*time.Second
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

	case "configuration_calculation_duration":
		summary.TSNCalcMs = metric.GetValue()
		summary.HasTSNCalc = true
		r.recentTSNCalc = &MetricSample{Timestamp: occurredAt, Value: metric.GetValue()}

	case "configuration_application_duration":
		summary.ConfigAppMs = metric.GetValue()
		summary.HasConfigApp = true
		r.recentConfigApp = &MetricSample{Timestamp: occurredAt, Value: metric.GetValue()}

	case "configuration_preparation_duration":
		summary.ConfigPrepMs = metric.GetValue()
		summary.HasConfigPrep = true
		r.recentConfigPrep = &MetricSample{Timestamp: occurredAt, Value: metric.GetValue()}

	case "configuration_commit_duration":
		summary.ConfigCommitMs = metric.GetValue()
		summary.HasConfigCommit = true
		r.recentConfigCommit = &MetricSample{Timestamp: occurredAt, Value: metric.GetValue()}
	}

	// If we received the total UNI response duration, record the summary row
	if summary.HasTotalUNI {
		r.writeSummaryRowLocked(summary)
	}
}

func (r *Receiver) writeSummaryRowLocked(s *LatencySummary) {
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
	s.WrittenToSummary = true

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
	if r.rawFile != nil {
		_ = r.rawFile.Sync()
		_ = r.rawFile.Close()
	}
	if r.summaryFile != nil {
		_ = r.summaryFile.Sync()
		_ = r.summaryFile.Close()
	}
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
