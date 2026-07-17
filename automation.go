package mmoney

import (
	"context"

	client "github.com/localitas/localitas-go"
)

const syncAutomationName = "MMoney: Sync"

func RegisterSyncAutomation(ctx context.Context, c *client.Client, appURL string) {
	if automationExists(ctx, c, syncAutomationName) {
		logger.Info("sync automation already registered")
		return
	}

	req := client.CreateAutomationRequest{
		Name:        syncAutomationName,
		Description: "Syncs financial data from MMoney every 10 minutes",
		DAGConfig: client.DAGConfig{
			DAGID:       "mmoney_sync",
			Name:        "MMoney: Sync",
			Description: "Calls the mmoney app sync endpoint",
			Nodes: []client.DAGNode{
				{
					NodeID:            "sync_all",
					NodeType:          "http-api",
					ExecutionStrategy: "raft-leader",
					Metadata: map[string]any{
						"url":                appURL + "/api/sync",
						"method":             "POST",
						"timeout_ms":         120000,
						"max_retries":        3,
						"backoff_ms":         5000,
						"backoff_multiplier": 2.0,
						"expected_status":    200,
					},
				},
			},
		},
		TriggerType: "periodic",
		TriggerConfig: client.TriggerConfig{
			Periodic: &client.PeriodicTrigger{
				Schedule:   "*/10 * * * *",
				Timezone:   "Local",
				MaxRetries: 2,
			},
		},
		IsEnabled: true,
	}

	if _, err := c.Automation().Create(ctx, req); err != nil {
		logger.Error("failed to register sync automation", "error", err)
		return
	}
	logger.Info("registered sync automation", "interval", "every 10 min")
}

func automationExists(ctx context.Context, c *client.Client, name string) bool {
	automations, err := c.Automation().List(ctx)
	if err != nil {
		return false
	}
	for _, a := range automations {
		if a.Name == name {
			return true
		}
	}
	return false
}
