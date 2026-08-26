package engine

import (
	"OpenCNC/common/structures/topology_config"
	protocolbackends "OpenCNC/config_service/pkg/protocolbackends"
	"context"
	"fmt"
	"sync"
)

type Operation struct {
	Config    *topology_config.NodeConfig
	Backend   protocolbackends.ProtocolBackend
	Prepared  bool
	Committed bool
}

type ConfigurationTransaction struct {
	MAX_CONCURRENT_CONFIGS int //determines the concurrent worker limit based on CPU cores
	ConfigId               string
	Operations             []Operation
}

func NewConfigurationTransaction(configId string) *ConfigurationTransaction {
	return &ConfigurationTransaction{
		ConfigId:               configId,
		MAX_CONCURRENT_CONFIGS: 1, // by default, execution is sequential
	}
}

func (t *ConfigurationTransaction) Prepare() error {

	err := t.executeConcurrently(func(ctx context.Context, i int) error {

		op := &t.Operations[i]

		if err := op.Backend.PrepareSnapshot(ctx, op.Config); err != nil {

			return fmt.Errorf(
				"prepare failed for node %s: %w",
				op.Backend.Name(),
				err,
			)
		}

		// Do not mark the operation as prepared if the transaction
		// was cancelled while the backend operation was running.
		if ctx.Err() != nil {
			op.Prepared = false
			return ctx.Err()
		}
		op.Prepared = true

		return nil
	})

	return err
}

func (t *ConfigurationTransaction) Commit() error {

	err := t.executeConcurrently(func(ctx context.Context, i int) error {

		op := &t.Operations[i]
		// Only commit operations that were successfully prepared.
		if !op.Prepared {
			return fmt.Errorf("operation was not prepared!")
		}

		if err := op.Backend.Commit(ctx); err != nil {

			return fmt.Errorf(
				"commit failed for node %s: %w",
				op.Backend.Name(),
				err,
			)
		}

		op.Committed = true
		op.Prepared = false

		return nil
	})

	// if there was an error during commit, attempt to rollback everything that was already committed.
	if err != nil {
		// All operations that were already started have finished.
		if rollbackErr := t.Rollback(); rollbackErr != nil {
			return fmt.Errorf("%w; rollback failed: %v", err, rollbackErr)
		}
		return fmt.Errorf("%w; transaction rolled back", err)
	}

	return nil
}

func (t *ConfigurationTransaction) Rollback() error {

	return t.executeConcurrently(func(ctx context.Context, i int) error {

		op := &t.Operations[i]
		if !op.Committed {
			return nil // nothing to rollback
		}

		if err := op.Backend.Rollback(ctx); err != nil {
			return fmt.Errorf("rollback failed for node %s: %w", op.Backend.Name(), err)
		}

		op.Committed = false
		return nil
	})
}

func (t *ConfigurationTransaction) executeConcurrently(fn func(ctx context.Context, index int) error) error {

	if t.MAX_CONCURRENT_CONFIGS < 1 {
		t.MAX_CONCURRENT_CONFIGS = 1
	}

	var wg sync.WaitGroup //wait for all operations to finish.
	var mu sync.Mutex     // safely store the first error.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	limiter := make(chan struct{}, t.MAX_CONCURRENT_CONFIGS) // restrict how many run simultaneously.

	var firstErr error

loop:
	for i := range t.Operations {

		// Stop starting new operations after cancellation.
		select {
		case limiter <- struct{}{}: // Acquire a concurrency slot (put empty value to channel).
		case <-ctx.Done(): // Stop starting new operations after cancellation.
			break loop
		}
		//limiter <- struct{}{}
		wg.Add(1) // mark the operation as started

		go func(index int) {
			defer wg.Done()              // mark the operation as finished
			defer func() { <-limiter }() // Release the concurrency slot.

			if err := fn(ctx, index); err != nil {
				mu.Lock()

				if firstErr == nil {
					firstErr = err
					cancel() // Cancel the context to stop starting new operations.
				}

				mu.Unlock()
			}
		}(i) // invoke the go routine with the operation index
	}

	wg.Wait() // Wait until all operations finish (the counter reaches zero).

	return firstErr
}
