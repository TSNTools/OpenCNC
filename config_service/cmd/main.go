package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"OpenCNC/common/observability"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	service "OpenCNC/config_service/grpc_server"
	"OpenCNC/config_service/pkg/engine" // Your wrapper implementing GNMIService
	_ "OpenCNC/config_service/pkg/plugins/netconf"
	_ "OpenCNC/config_service/pkg/protocolbackends/netconf"
	// Official gNMI package
)

func main() {
	startResourceMonitor()

	obsClient, err := observability.NewFromEnv("config-service")
	if err != nil {
		log.Fatalf("Observability init failed: %v", err)
	}
	if obsClient != nil {
		defer func() {
			_ = obsClient.Close()
		}()

		startupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = obsClient.EmitHealthStarted(startupCtx, "config-service-startup", "config-service started")
	}

	// --- Load server certificate and key ---
	/*
		serverCert, err := tls.LoadX509KeyPair("/certs/tls.crt", "/certs/tls.key")
		if err != nil {
			obsClient.FatalF("Failed to load server TLS cert/key: %v", err)
		}

		// --- Load CA certificate to verify clients ---
		caCertPEM, err := os.ReadFile("/certs/ca.crt")
		if err != nil {
			obsClient.FatalF("Failed to load CA certificate: %v", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCertPEM) {
			obsClient.FatalF("Failed to append CA certificate to pool")
		}

		// --- Configure mutual TLS ---
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{serverCert}, // server identity
			ClientCAs:    caCertPool,                    // trust clients signed by this CA
			ClientAuth:   tls.NoClientCert,              // only server authenticates
			// //tls.RequireAndVerifyClientCert, // 🔒 require valid client cert
			MinVersion: tls.VersionTLS13, // enforce modern TLS
		}

		creds := credentials.NewTLS(tlsConfig)
	*/
	// --- Create TCP listener ---
	listener, err := net.Listen("tcp", ":5150")
	if err != nil {
		obsClient.FatalF("Failed to listen on :5150: %v", err)
	}

	// --- Create gRPC server with TLS credentials ---
	//grpcServer := grpc.NewServer(grpc.Creds(creds))
	grpcServer := grpc.NewServer()
	//logger.Println("Starting gRPC server without TLS (for testing)...")

	// --- Create the configuration engine and register backends ---
	engine := engine.NewMappingEngine(obsClient)

	// --- Register ConfigService and gNMI service ---
	svc := service.NewConfigServiceServerImpl(obsClient, engine)
	service.RegisterConfigServiceServer(grpcServer, svc)

	//gnmi.RegisterGNMIServer(grpcServer, gnmiImpl.NewGNMIService(logger))

	// --- Optional: reflection ---
	reflection.Register(grpcServer)

	if obsClient != nil {
		obsClient.Println("gRPC server with TLS started on port 5150")
	}

	// --- Start serving ---
	if err := grpcServer.Serve(listener); err != nil {
		if obsClient != nil {
			errCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = obsClient.EmitHealthError(errCtx, "config-service-serve", err.Error(), "")
		}
		obsClient.FatalF("gRPC server failed: %v", err)
	}
}

// startResourceMonitor periodically samples goroutines, CPU %, and memory every 100ms and writes to CSV.
func startResourceMonitor() {
	csvPath := os.Getenv("CONFIG_SERVICE_RESOURCE_CSV")
	if csvPath == "" {
		csvPath = "config_service_resources.csv"
	}

	interval := 100 * time.Millisecond
	if envInterval := os.Getenv("CONFIG_SERVICE_RESOURCE_INTERVAL_MS"); envInterval != "" {
		if ms, err := strconv.Atoi(envInterval); err == nil && ms > 0 {
			interval = time.Duration(ms) * time.Millisecond
		}
	}

	fileExisted := false
	if fi, err := os.Stat(csvPath); err == nil && fi.Size() > 0 {
		fileExisted = true
	}

	f, err := os.OpenFile(csvPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("[METRICS WARN] Failed to open resource CSV %s: %v", csvPath, err)
		return
	}

	writer := csv.NewWriter(f)
	if !fileExisted {
		headers := []string{
			"timestamp",
			"goroutines",
			"cpu_pct",
			"rss_mem_mb",
			"vms_mem_mb",
			"heap_alloc_mb",
			"heap_sys_mb",
			"num_gc",
		}
		_ = writer.Write(headers)
		writer.Flush()
	}

	log.Printf("[METRICS] Resource monitor started (interval: %v, output: %s)", interval, csvPath)

	go func() {
		defer f.Close()
		defer writer.Flush()

		var prevRusage syscall.Rusage
		_ = syscall.Getrusage(syscall.RUSAGE_SELF, &prevRusage)
		prevCPUTime := time.Duration(prevRusage.Utime.Sec)*time.Second + time.Duration(prevRusage.Utime.Usec)*time.Microsecond +
			time.Duration(prevRusage.Stime.Sec)*time.Second + time.Duration(prevRusage.Stime.Usec)*time.Microsecond
		prevTime := time.Now()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		var m runtime.MemStats

		for range ticker.C {
			now := time.Now()
			dt := now.Sub(prevTime)

			// CPU % from process CPU time
			var ru syscall.Rusage
			cpuPct := 0.0
			if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err == nil {
				currCPUTime := time.Duration(ru.Utime.Sec)*time.Second + time.Duration(ru.Utime.Usec)*time.Microsecond +
					time.Duration(ru.Stime.Sec)*time.Second + time.Duration(ru.Stime.Usec)*time.Microsecond
				if dt > 0 {
					dCPU := currCPUTime - prevCPUTime
					cpuPct = (float64(dCPU) / float64(dt)) * 100.0
				}
				prevCPUTime = currCPUTime
			}
			prevTime = now

			// Goroutines
			numGoroutines := runtime.NumGoroutine()

			// Go Runtime Memory (Heap & GC)
			runtime.ReadMemStats(&m)
			heapAllocMB := float64(m.Alloc) / (1024 * 1024)
			heapSysMB := float64(m.Sys) / (1024 * 1024)
			numGC := m.NumGC

			// OS Resident & Virtual Memory
			rssMB, vmsMB := readProcSelfMemory()

			record := []string{
				now.Format(time.RFC3339Nano),
				strconv.Itoa(numGoroutines),
				fmt.Sprintf("%.2f", cpuPct),
				fmt.Sprintf("%.2f", rssMB),
				fmt.Sprintf("%.2f", vmsMB),
				fmt.Sprintf("%.2f", heapAllocMB),
				fmt.Sprintf("%.2f", heapSysMB),
				strconv.FormatUint(uint64(numGC), 10),
			}

			_ = writer.Write(record)
			writer.Flush()
		}
	}()
}

func readProcSelfMemory() (float64, float64) {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0, 0
	}
	var rssMB, vmsMB float64
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if kb, err := strconv.ParseFloat(fields[1], 64); err == nil {
					rssMB = kb / 1024.0
				}
			}
		} else if strings.HasPrefix(line, "VmSize:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if kb, err := strconv.ParseFloat(fields[1], 64); err == nil {
					vmsMB = kb / 1024.0
				}
			}
		}
	}
	return rssMB, vmsMB
}

