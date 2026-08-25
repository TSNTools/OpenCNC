package uni

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	handler "main-service/pkg/event-handler"
	"main-service/pkg/structures/configuration"
	"main-service/pkg/structures/streamObjects"

	//	"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"
	"github.com/go-openapi/runtime/middleware/header"
	"github.com/gogo/protobuf/jsonpb"
)

const PORT uint16 = 8080

//var log = logger.GetLogger()

func StartServer() {
	//log.Infof("Starting UNI server")
	fmt.Println("Starting UNI server")
	http.HandleFunc("/add_stream", addStream)
	//	http.HandleFunc("/update_stream", updateStream)
	//	http.HandleFunc("/remove_stream", removeStream)
	//	http.HandleFunc("/join_stream", joinStream)
	//	http.HandleFunc("/leave_stream", leaveStream)

	http.HandleFunc("/register_node", registerNode)

	//log.Info("TODO: Add all endpoints so that a user could see them (check what endpoints AccessTSN expects)")
	//log.Infof("API endpoint -> http://localhost:%d/add_stream", PORT)
	// log.Infof("API endpoint -> http://localhost:%d/update_stream", PORT)
	// log.Infof("API endpoint -> http://localhost:%d/remove_stream", PORT)
	// log.Infof("API endpoint -> http://localhost:%d/join_stream", PORT)
	// log.Infof("API endpoint -> http://localhost:%d/leave_stream", PORT)
	//log.Infof("API endpoint -> http://localhost:%d/register_node", PORT)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", PORT), nil); err != nil {
		//log.Errorf("Failed to listen and server on %d, with error: %+v", PORT, err)
		fmt.Printf("Failed to listen and server on %d, with error: %+v", PORT, err)

	}
}

func addStream(writer http.ResponseWriter, req *http.Request) {
	timeOfReq := time.Now()

	if err := checkHeader(req); err != nil {
		http.Error(writer, err.Error(), http.StatusUnsupportedMediaType)
		return
	}

	//log.Info("Received add_stream request")
	fmt.Println("Received add_stream request")

	var configRequest configuration.ConfigRequest

	err := jsonpb.Unmarshal(req.Body, &configRequest)

	// NEED A REMAKE TO SUIT PROTO UMARSHSALING ERRORS
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnsupportedTypeError

		switch {
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
			http.Error(writer, msg, http.StatusBadRequest)
		case errors.Is(err, io.ErrUnexpectedEOF):
			msg := "Request body contains badly-formed JSON"
			http.Error(writer, msg, http.StatusBadRequest)
		case errors.As(err, &unmarshalTypeError):
			// msg := fmt.Sprintf("Request body contains invalid value for the %q field (at position %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)
			msg := "Request body contains invalid structure"
			http.Error(writer, msg, http.StatusBadRequest)
		case strings.HasPrefix(err.Error(), "json:unknown field"):
			fieldName := strings.TrimPrefix(err.Error(), "json:unknown field")
			msg := fmt.Sprintf("Request body contains unknown field %s", fieldName)
			http.Error(writer, msg, http.StatusBadRequest)
		case errors.Is(err, io.EOF):
			msg := "Request body must not be empty"
			http.Error(writer, msg, http.StatusBadRequest)
		case err.Error() == "http: request body too large":
			msg := "Request body must not be larger than 1MB"
			http.Error(writer, msg, http.StatusRequestEntityTooLarge)
		default:
			//log.Error(err.Error())
			fmt.Println(err.Error())

			http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	fmt.Println("Valid Request")
	// log.Infof("%+v", reflect.TypeOf(req.Body))

	// Call handler to deal with addStream request
	confId, err := handler.HandleAddStreamEvent(&configRequest, timeOfReq)
	if err != nil {
		//log.Errorf("Failed handling event: %v", err)
		fmt.Printf("Failed handling event: %v", err)
		http.Error(writer, "Failed handling event.", http.StatusBadRequest)
		return
	}

	// TODO: Split response into the different status groups and send them to each endstation in the stream
	// Requires: split the response before it is serialized and instead return a list of byte slices
	resp, err := createResponse(confId, &configRequest)
	if err != nil {
		//log.Errorf("Failed to create UNI response!")
		fmt.Println("Failed to create UNI response!")
		return
	}

	writer.Header().Add("Content-Type", "application/json; charset=utf-8")
	writer.Write(resp)
}

func updateStream(writer http.ResponseWriter, req *http.Request) {
	//log.Info("TODO: Implement functionality")
	if err := checkHeader(req); err != nil {
		http.Error(writer, err.Error(), http.StatusUnsupportedMediaType)
		return
	}

	writer.Write([]byte("Done!"))
}

func removeStream(writer http.ResponseWriter, req *http.Request) {
	//log.Info("TODO: Implement functionality")
	if err := checkHeader(req); err != nil {
		http.Error(writer, err.Error(), http.StatusUnsupportedMediaType)
		return
	}
}

func joinStream(writer http.ResponseWriter, req *http.Request) {
	//log.Info("TODO: Implement functionality")
	if err := checkHeader(req); err != nil {
		http.Error(writer, err.Error(), http.StatusUnsupportedMediaType)
		return
	}
}

func leaveStream(writer http.ResponseWriter, req *http.Request) {
	//log.Info("TODO: Implement functionality")
	if err := checkHeader(req); err != nil {
		http.Error(writer, err.Error(), http.StatusUnsupportedMediaType)
		return
	}
}

func registerNode(writer http.ResponseWriter, req *http.Request) {
	if err := checkHeader(req); err != nil {
		http.Error(writer, err.Error(), http.StatusUnsupportedMediaType)
		return
	}

	// log.Infof("Requests body looks like: %v", req.Body)
	fmt.Printf("Requests body looks like: %v", req.Body)

	var nodeObj streamObjects.NodeObject
	err := jsonpb.Unmarshal(req.Body, &nodeObj)

	if err != nil {
		//log.Errorf("Failed to read req.Body: %v", err)
		http.Error(writer, "Failed to read body", http.StatusBadRequest)
		return
	}

	handler.RegisterNode(&nodeObj)
}

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
