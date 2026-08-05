package job

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MrityunjayRoy/prod-go-boilerplate/internal/config"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
)

var emailClient *email.Client

func (j *JobService) InitHandlers(cfg *config.Config, logger *zerolog.Logger) {
	emailClient = email.NewClient(config, logger)
}

func (j *JobService) handleWelcomeEmailTask(ctx context.Context, t *asynq.Task) error {
	var p WelcomeEmailPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("Failed to Unmarshal welcome email payload: %w", err)
	}

	j.logger.Info().
		Str("type", "welcome").
		Str("to", p.To).
		Msg("Processing welcome email task")

	err := emailClient.SendWelcomeEmail(p.To, p.FirstName)
	if err != nil {
		j.logger.Error().
			Str("type", "welcome").
			Str("to", p.To).
			Err(err).
			Msg("Failed to send welcome email")
		return err
	}
	j.logger.Info().
		Str("type", "welcome").
		Str("to", p.To).
		Msg("Failed to send welcome email")

	return nil
}
