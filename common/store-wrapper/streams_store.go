package storewrapper

import (
	"OpenCNC/common/structures/stream"
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"
)

func StoreStream(streamObj *stream.Stream, id string) error {
	if streamObj == nil {
		return fmt.Errorf("cannot store nil stream")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("stream ID cannot be empty")
	}

	obj, err := proto.Marshal(streamObj)
	if err != nil {
		return fmt.Errorf(
			"failed to marshal stream %s: %w",
			id,
			err,
		)
	}

	urn := "streams." + id

	if err := SendToStore(obj, urn); err != nil {
		return fmt.Errorf(
			"failed to store stream %s: %w",
			id,
			err,
		)
	}

	return nil
}

func GetStreamByID(id string) (*stream.Stream, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("stream ID cannot be empty")
	}

	rawData, err := getFromStoreWithPrefix("streams." + id)
	if err != nil {
		return nil, err
	}

	if len(rawData.Kvs) == 0 {
		return nil, fmt.Errorf("stream %s not found", id)
	}

	streamObj := &stream.Stream{}

	if err := proto.Unmarshal(
		[]byte(rawData.Kvs[0].Value),
		streamObj,
	); err != nil {
		return nil, fmt.Errorf(
			"failed unmarshaling stream %s: %w",
			id,
			err,
		)
	}

	return streamObj, nil
}

func DeleteStream(id string) error {
	id = strings.TrimSpace(id)

	if id == "" {
		return fmt.Errorf("stream ID cannot be empty")
	}

	client, err := createEtcdClient()
	if err != nil {
		return err
	}
	defer client.Close()

	key := "streams/" + id

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf(
			"failed to delete stream %s: %w",
			id,
			err,
		)
	}

	return nil
}

func GetStreams() ([]*stream.Stream, error) {
	rawData, err := getFromStoreWithPrefix("streams")
	if err != nil {
		return nil, err
	}

	streams := make([]*stream.Stream, 0, len(rawData.Kvs))

	for _, rawStream := range rawData.Kvs {
		streamObj := &stream.Stream{}

		if err := proto.Unmarshal(rawStream.Value, streamObj); err != nil {
			log.Errorf("failed to unmarshal stream: %v", err)
			continue
		}

		streams = append(streams, streamObj)
	}

	return streams, nil
}
