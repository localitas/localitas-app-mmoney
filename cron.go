package mmoney

import (
	"net/http"

	client "github.com/localitas/localitas-go"
)

var cronSpec = client.CronSpec{
	Jobs: []client.CronJob{
		{
			ID:          "cron:mmoney:sync",
			Path:        "/api/sync",
			Method:      "POST",
			Schedule:    "*/10 * * * *",
			Description: "Syncs financial data",
			Timeout:     "120s",
			Retry: &client.CronRetry{
				MaxAttempts:       3,
				InitialDelay:      "5s",
				MaxDelay:          "30s",
				Backoff:           "exponential",
				BackoffMultiplier: 2.0,
			},
		},
	},
}

func HandleCron(w http.ResponseWriter, r *http.Request) {
	cronSpec.ServeHTTP(w, r)
}
