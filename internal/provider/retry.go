// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// RetryConfig defines parameters for retrying an operation.
type RetryConfig struct {
	MaxRetries    int
	TotalWaitTime time.Duration
	RetryDelay    time.Duration
}

// Retry executes the provided operation function with retry logic.
// The operation function should return (true, nil) when successful.
func Retry(ctx context.Context, cfg RetryConfig, operation func() (bool, error)) error {
	startTime := time.Now()
	var lastErr error
	for i := 0; i < cfg.MaxRetries; i++ {
		if time.Since(startTime) > cfg.TotalWaitTime {
			tflog.Warn(ctx, "Exceeded total retry wait time", map[string]interface{}{
				"max_wait_seconds": cfg.TotalWaitTime.Seconds(),
			})
			break
		}
		done, err := operation()
		if done {
			return nil
		}
		lastErr = err
		tflog.Debug(ctx, "Retrying operation", map[string]interface{}{
			"retry": i + 1,
		})
		time.Sleep(cfg.RetryDelay)
	}
	return lastErr
}
