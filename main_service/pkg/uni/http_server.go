package uni_server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"OpenCNC/common/observability"
	"OpenCNC/common/structures/topology"
	"OpenCNC/common/structures/uni"
	handler "OpenCNC/main_service/pkg/event-handler"

	"github.com/go-openapi/runtime/middleware/header"
	"github.com/gogo/protobuf/jsonpb"
	"google.golang.org/protobuf/encoding/protojson"
)

// StartHttpServer starts the UNI HTTP server.
func StartHttpServer(
	ctx context.Context,
	port uint16,
	obs *observability.Client,
) {
	if obs != nil {
		_ = obs.Info(ctx, "Starting UNI HTTP server")
	}

	http.HandleFunc("/add_stream", func(writer http.ResponseWriter, req *http.Request) {
		addStream(obs, writer, req)
	})

	// http.HandleFunc("/update_stream", updateStream)
	// http.HandleFunc("/remove_stream", removeStream)
	// http.HandleFunc("/join_stream", joinStream)
	// http.HandleFunc("/leave_stream", leaveStream)

	http.HandleFunc("/register_node", func(writer http.ResponseWriter, req *http.Request) {
		registerNode(obs, writer, req)
	})

	// TODO: Add all endpoints so that a user could see them
	// (check what endpoints AccessTSN expects)
	// API endpoint -> http://localhost:%d/add_stream
	// API endpoint -> http://localhost:%d/update_stream
	// API endpoint -> http://localhost:%d/remove_stream
	// API endpoint -> http://localhost:%d/join_stream
	// API endpoint -> http://localhost:%d/leave_stream
	// API endpoint -> http://localhost:%d/register_node

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed to listen and serve on %d, with error: %v",
				port,
				err,
			))
		}
	}
}

func addStream(
	obs *observability.Client,
	writer http.ResponseWriter,
	req *http.Request,
) {
	ctx := req.Context()
	timeOfReq := time.Now()

	if err := checkHeader(req); err != nil {
		http.Error(writer, err.Error(), http.StatusUnsupportedMediaType)
		return
	}

	if obs != nil {
		_ = obs.Info(ctx, "Received add_stream request")
	}

	var configRequest uni.ConfigRequest

	err := jsonpb.Unmarshal(req.Body, &configRequest)

	// NEED A REMAKE TO SUIT PROTO UMARSHSALING ERRORS
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnsupportedTypeError

		switch {
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf(
				"Request body contains badly-formed JSON (at position %d)",
				syntaxError.Offset,
			)
			http.Error(writer, msg, http.StatusBadRequest)

		case errors.Is(err, io.ErrUnexpectedEOF):
			msg := "Request body contains badly-formed JSON"
			http.Error(writer, msg, http.StatusBadRequest)

		case errors.As(err, &unmarshalTypeError):
			// msg := fmt.Sprintf(
			//     "Request body contains invalid value for the %q field (at position %d)",
			//     unmarshalTypeError.Field,
			//     unmarshalTypeError.Offset,
			// )
			msg := "Request body contains invalid structure"
			http.Error(writer, msg, http.StatusBadRequest)

		case strings.HasPrefix(err.Error(), "json:unknown field"):
			fieldName := strings.TrimPrefix(
				err.Error(),
				"json:unknown field",
			)
			msg := fmt.Sprintf(
				"Request body contains unknown field %s",
				fieldName,
			)
			http.Error(writer, msg, http.StatusBadRequest)

		case errors.Is(err, io.EOF):
			msg := "Request body must not be empty"
			http.Error(writer, msg, http.StatusBadRequest)

		case err.Error() == "http: request body too large":
			msg := "Request body must not be larger than 1MB"
			http.Error(writer, msg, http.StatusRequestEntityTooLarge)

		default:
			if obs != nil {
				_ = obs.Error(ctx, err.Error())
			}

			http.Error(
				writer,
				http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError,
			)
		}

		return
	}

	if obs != nil {
		_ = obs.Info(ctx, "Valid Request")
	}

	// TODO: Split response into the different status groups
	// and send them to each endstation in the stream.
	// Requires: split the response before it is serialized
	// and instead return a list of byte slices.

	// Call handler to deal with addStream request
	confId, err := handler.HandleAddStreamEvent(
		ctx,
		obs,
		&configRequest,
		timeOfReq,
	)
	if err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed handling event: %v",
				err,
			))
		}

		http.Error(
			writer,
			"Failed handling event.",
			http.StatusBadRequest,
		)
		return
	}

	response, err := createResponse(
		ctx,
		obs,
		confId,
		&configRequest,
	)
	if err != nil {
		if obs != nil {
			_ = obs.Error(ctx, "Failed to create UNI response!")
		}

		return
	}

	resp, err := protojson.Marshal(response)
	if err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed to marshal UNI ConfigResponse: %v",
				err,
			))
		}

		return
	}

	writer.Header().Add(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	_, _ = writer.Write(resp)
}

func updateStream(
	writer http.ResponseWriter,
	req *http.Request,
) {
	// TODO: Implement functionality
	if err := checkHeader(req); err != nil {
		http.Error(
			writer,
			err.Error(),
			http.StatusUnsupportedMediaType,
		)
		return
	}

	_, _ = writer.Write([]byte("Done!"))
}

func removeStream(
	writer http.ResponseWriter,
	req *http.Request,
) {
	// TODO: Implement functionality
	if err := checkHeader(req); err != nil {
		http.Error(
			writer,
			err.Error(),
			http.StatusUnsupportedMediaType,
		)
		return
	}
}

func joinStream(
	writer http.ResponseWriter,
	req *http.Request,
) {
	// TODO: Implement functionality
	if err := checkHeader(req); err != nil {
		http.Error(
			writer,
			err.Error(),
			http.StatusUnsupportedMediaType,
		)
		return
	}
}

func leaveStream(
	writer http.ResponseWriter,
	req *http.Request,
) {
	// TODO: Implement functionality
	if err := checkHeader(req); err != nil {
		http.Error(
			writer,
			err.Error(),
			http.StatusUnsupportedMediaType,
		)
		return
	}
}

func registerNode(
	obs *observability.Client,
	writer http.ResponseWriter,
	req *http.Request,
) {
	ctx := req.Context()

	if err := checkHeader(req); err != nil {
		http.Error(
			writer,
			err.Error(),
			http.StatusUnsupportedMediaType,
		)
		return
	}

	if obs != nil {
		_ = obs.Info(ctx, fmt.Sprintf(
			"Requests body looks like: %v",
			req.Body,
		))
	}

	var nodeObj topology.Node

	err := jsonpb.Unmarshal(req.Body, &nodeObj)
	if err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed to read req.Body: %v",
				err,
			))
		}

		http.Error(
			writer,
			"Failed to read body",
			http.StatusBadRequest,
		)
		return
	}

	handler.RegisterNode(ctx, obs, &nodeObj)
}

func checkHeader(req *http.Request) error {
	if req.Header.Get("Content-Type") != "" {
		value, _ := header.ParseValueAndParams(
			req.Header,
			"Content-Type",
		)

		if value != "application/json" {
			msg := "Content-Type header is not application/json"
			return errors.New(msg)
		}
	}

	return nil
}
