package nni

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	store "main-service/pkg/store-wrapper"
	devicemodelregistry "main-service/pkg/structures/device-model-registry"
	"main-service/pkg/structures/topology"

	//"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"

	"github.com/go-openapi/runtime/middleware/header"
	"google.golang.org/protobuf/encoding/protojson"
)

//var log = logger.GetLogger()

const PORT uint16 = 8000

type NodeValidationError struct {
	Name        string `json:"name"`
	DeviceModel string `json:"deviceModel,omitempty"`
	Reason      string `json:"reason"`
}

type UploadWrapper struct {
	Models []struct {
		Name      string `json:"model-name"`
		YangFiles []struct {
			Name      string `json:"file-name"`
			Revision  string `json:"file-revision"`
			Structure string `json:"structure"`
		} `json:"yang-files"`
	} `json:"models"`
}

func StartServer() {
	fmt.Println("Starting NNI server")

	// Serve dashboard.html at root
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web-interface/dashboard.html")
	})

	// API endpoints
	http.HandleFunc("/add_device_model", addDeviceModel)
	http.HandleFunc("/add_topology", addTopology)
	http.HandleFunc("/extend_topology", extendTopology)
	http.HandleFunc("/get_topology", getTopology)
	http.HandleFunc("/get_device_models", getDeviceModels)
	// ... other endpoints ...

	// Serve static files (CSS, JS) correctly
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web-interface"))))

	if err := http.ListenAndServe(fmt.Sprintf(":%d", PORT), nil); err != nil {
		fmt.Printf("Failed to listen and serve on %d: %v\n", PORT, err)
	}
}

func addDeviceModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Unmarshal JSON into plain Go wrapper
	var wrapper UploadWrapper
	if err := json.Unmarshal(body, &wrapper); err != nil {
		http.Error(w, "failed to parse JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(wrapper.Models) == 0 {
		http.Error(w, "no device models in request", http.StatusBadRequest)
		return
	}

	stored := 0
	skipped := []string{}

	for _, m := range wrapper.Models {
		if m.Name == "" {
			skipped = append(skipped, "<empty-name>")
			continue
		}

		// Convert each plain Go model to protobuf DeviceModel
		model := &devicemodelregistry.DeviceModel{
			Name: m.Name,
		}

		for _, y := range m.YangFiles {
			if y.Name == "" {
				fmt.Printf("Skipping YangFile with empty name in model %s\n", m.Name)
				continue
			}
			model.YangFiles = append(model.YangFiles, &devicemodelregistry.YangFile{
				Name:      y.Name,
				Revision:  y.Revision,
				Structure: y.Structure,
			})
		}

		// Store in etcd
		err := store.StoreDeviceModel(model)
		if err != nil {
			skipped = append(skipped, m.Name)
			fmt.Printf("Failed to store model %s: %v\n", m.Name, err)
			continue
		}
		stored++
	}

	// Return JSON response to JS
	resp := map[string]interface{}{
		"status":        "ok",
		"message":       fmt.Sprintf("%d device model(s) stored successfully!", stored),
		"stored":        stored,
		"skippedModels": skipped,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func addTopology(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	// Only allow POST
	if req.Method != http.MethodPost {
		http.Error(w, `{"status":"error","message":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Must be JSON
	if req.Header.Get("Content-Type") != "application/json" {
		http.Error(w, `{"status":"error","message":"Unsupported Media Type, expected application/json"}`, http.StatusUnsupportedMediaType)
		return
	}

	// Read body
	data, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, `{"status":"error","message":"Failed to read request body"}`, http.StatusBadRequest)
		return
	}

	var topo topology.Topology

	// Unmarshal JSON into protobuf safely
	opts := protojson.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: true,
	}
	if err := opts.Unmarshal(data, &topo); err != nil {
		http.Error(w, fmt.Sprintf(`{"status":"error","message":"Failed to parse topology: %v"}`, err), http.StatusBadRequest)
		return
	}

	// Print topology
	topo.Print()

	invalidNodes, err := verifyDeviceModelsExist(&topo)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":       "error",
			"message":      "Some nodes have missing or invalid device models",
			"invalidNodes": invalidNodes,
		})
		return
	}

	// Store topology
	if err := store.StoreTopology(&topo); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("Failed to store topology: %v", err),
		})
		return
	}

	// Success response
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Topology stored successfully",
	})
}

// TODO: do it! update topology
func extendTopology(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	// Only allow POST
	if req.Method != http.MethodPost {
		http.Error(w, `{"status":"error","message":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Must be JSON
	if req.Header.Get("Content-Type") != "application/json" {
		http.Error(w, `{"status":"error","message":"Unsupported Media Type, expected application/json"}`, http.StatusUnsupportedMediaType)
		return
	}

	// Read body
	data, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, `{"status":"error","message":"Failed to read request body"}`, http.StatusBadRequest)
		return
	}

	var topo topology.Topology

	// Unmarshal JSON into protobuf safely
	opts := protojson.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: true,
	}
	if err := opts.Unmarshal(data, &topo); err != nil {
		http.Error(w, fmt.Sprintf(`{"status":"error","message":"Failed to parse topology: %v"}`, err), http.StatusBadRequest)
		return
	}

	// Print topology
	topo.Print()

	invalidNodes, err := verifyDeviceModelsExist(&topo)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":       "error",
			"message":      "Some nodes have missing or invalid device models",
			"invalidNodes": invalidNodes,
		})
		return
	}

	// Store topology
	if err := store.StoreTopology(&topo); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("Failed to store topology: %v", err),
		})
		return
	}

	// Success response
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Topology stored successfully",
	})
}

// GetTalkers returns all talker nodes
/*func getTalkers(w http.ResponseWriter, r *http.Request) {
	nodes, err := store.GetNodesByRole(topology.END_STATION) // or store.GetTalkers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, nodes)
}

// GetListeners returns all listener nodes
func getListeners(w http.ResponseWriter, r *http.Request) {
	nodes, err := store.GetNodesByRole(topology.END_STATION) // if listeners differ, adjust
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, nodes)
}

// GetSwitches returns all bridges
func getSwitches(w http.ResponseWriter, r *http.Request) {
	nodes, err := store.GetNodesByRole(topology.BRIDGE)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, nodes)
}

// GetStreams returns all streams (assuming you have a store function)
func getStreams(w http.ResponseWriter, r *http.Request) {
	streams, err := store.GetStreams()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, streams)
} */

// GetTopology returns the full topology
func getTopology(w http.ResponseWriter, r *http.Request) {
	topo, err := store.GetTopology()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, topo)
}

func getDeviceModels(w http.ResponseWriter, r *http.Request) {
	models, err := store.GetDeviceModelRegistry()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Filter out empty models
	var validModels []*devicemodelregistry.DeviceModel
	for _, m := range models.DeviceModels {
		if m.Name != "" {
			validModels = append(validModels, m)
		}
	}

	writeJSON(w, validModels)
}

// Helper to write JSON with proper headers
func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(data)
}

// TODO: check the extend topology - need to make an handler..
func addStream(writer http.ResponseWriter, req *http.Request) {
	if err := checkHeader(req); err != nil {
		http.Error(writer, err.Error(), http.StatusUnsupportedMediaType)
		return
	}
	//TODO: Implement logic using bridge.proto
}

/////

func checkHeader(req *http.Request) error {
	if req.Header.Get("Content-Type") != "" {
		value, _ := header.ParseValueAndParams(req.Header, "Content-Type")
		if value != "application/json" {
			msg := "Content-Type header is not application/json"
			return errors.New(msg)
		}
	}

	return nil
}

func verifyDeviceModelsExist(topo *topology.Topology) ([]NodeValidationError, error) {
	var invalidNodes []NodeValidationError

	for _, node := range topo.Nodes {
		deviceModel := node.DeviceInfo.GetDeviceModel()

		if deviceModel == "" {
			invalidNodes = append(invalidNodes, NodeValidationError{
				Name:   node.Name,
				Reason: "no device model specified",
			})
			continue
		}

		_, err := store.GetDeviceModel(deviceModel)
		if err != nil {
			invalidNodes = append(invalidNodes, NodeValidationError{
				Name:        node.Name,
				DeviceModel: deviceModel,
				Reason:      "device model not found in etcd",
			})
		}
	}

	if len(invalidNodes) > 0 {
		return invalidNodes, fmt.Errorf("some nodes have missing or invalid device models")
	}

	return nil, nil
}
