package buildledger

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/sreejay-reddy/odyssey/odyssey-go/types"
	"github.com/sreejay-reddy/odyssey/odyssey-go/configutil"
	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/registry"
)

func BuildLedger(ctx context.Context, conn *pgx.Conn, cfg configutil.Config, key string, steps []types.Step) (bool, error){

	if key == "" {
		return false, errors.New("key cannot be empty")
	}

	if len(steps) == 0 {
		return false, errors.New("build ledger requires at least one step")
	}

	targets := make(map[string]struct{})

	for _, step := range steps {
		if _, exists := targets[step.Target]; exists {
			return false, errors.New("step targets must be unique")
		}

		targets[step.Target] = struct{}{}
	}

	for _, step := range steps {
		if strings.TrimSpace(step.Target) == "" {
			return false, errors.New("target cannot be empty")
		}

		_ , exists := registry.Get(step.Target)

		if !exists && step.Delegate == "" {
			return false, errors.New("target doesn't exist in registry and target isn't delegated")
		}

		if step.Delegate != "" {
			if _, exists := cfg.Services[step.Delegate]; !exists {
				return false, errors.New(
					"delegate does not exist in configured services",
				)
			}
		}
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	ledgerRows := make([][]any, 0, len(steps))
	deliveryRows := make([][]any, 0)

	for _, step := range steps {
		var input any

		if step.Input != nil {
			inputJSON, err := json.Marshal(step.Input)

			if err != nil {
				return false, err
			}
			input = inputJSON
		}

		mode := "local"

		if step.Delegate != "" {
			mode = "delegated"

			deliveryRows = append(deliveryRows, []any{
				key,
				step.Target,
				step.Delegate,
			})
		}

		ledgerRows = append(ledgerRows, []any{
			key,
			step.Target,
			mode,
			input,
		})
	}

	_, err = tx.CopyFrom(
		ctx,
		pgx.Identifier{"odyssey_ledger"},
		[]string{
			"key",
			"target",
			"mode",
			"input",
		},
			pgx.CopyFromRows(ledgerRows),
	)

	if err != nil {
		return false, err
	}

	if len(deliveryRows) > 0 {
		_, err = tx.CopyFrom(
			ctx,
			pgx.Identifier{"odyssey_deliveries"},
			[]string{
				"key",
				"target",
				"emit_to",
			},
			pgx.CopyFromRows(deliveryRows),
		)

		if err != nil {
			return false, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}

	return true, nil
}