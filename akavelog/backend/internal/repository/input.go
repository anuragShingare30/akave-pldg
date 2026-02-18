package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akave-ai/akavelog/internal/model"
)

// InputRepository persists and reads input definitions.
type InputRepository struct {
	pool *pgxpool.Pool
}

// NewInputRepository returns an InputRepository using the given pool.
func NewInputRepository(pool *pgxpool.Pool) *InputRepository {
	return &InputRepository{pool: pool}
}

// Create inserts a new input and returns it with ID and CreatedAt set.
func (r *InputRepository) Create(ctx context.Context, input *model.Input) error {
	query := `
		INSERT INTO inputs (id, type, title, configuration, global, node_id, creator_user_id, desired_state)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at`
	if input.ID == uuid.Nil {
		input.ID = uuid.New()
	}
	return r.pool.QueryRow(ctx, query,
		input.ID,
		input.Type,
		input.Title,
		input.Configuration,
		input.Global,
		input.NodeID,
		input.CreatorUserID,
		input.DesiredState,
	).Scan(&input.ID, &input.CreatedAt)
}

// List returns all inputs ordered by created_at descending.
func (r *InputRepository) List(ctx context.Context) ([]model.Input, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, type, title, configuration, global, node_id, creator_user_id, created_at, desired_state
		FROM inputs
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.Input
	for rows.Next() {
		var in model.Input
		if err := rows.Scan(
			&in.ID,
			&in.Type,
			&in.Title,
			&in.Configuration,
			&in.Global,
			&in.NodeID,
			&in.CreatorUserID,
			&in.CreatedAt,
			&in.DesiredState,
		); err != nil {
			return nil, err
		}
		list = append(list, in)
	}
	return list, rows.Err()
}

// GetByID returns one input by id, or nil if not found.
func (r *InputRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Input, error) {
	var in model.Input
	err := r.pool.QueryRow(ctx, `
		SELECT id, type, title, configuration, global, node_id, creator_user_id, created_at, desired_state
		FROM inputs WHERE id = $1`, id).Scan(
		&in.ID,
		&in.Type,
		&in.Title,
		&in.Configuration,
		&in.Global,
		&in.NodeID,
		&in.CreatorUserID,
		&in.CreatedAt,
		&in.DesiredState,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &in, nil
}

// Update updates an existing input by id. Only type, title, configuration, and desired_state are updated.
func (r *InputRepository) Update(ctx context.Context, input *model.Input) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE inputs SET type = $1, title = $2, configuration = $3, desired_state = $4
		WHERE id = $5`,
		input.Type,
		input.Title,
		input.Configuration,
		input.DesiredState,
		input.ID,
	)
	return err
}

// Delete removes an input by id.
func (r *InputRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM inputs WHERE id = $1`, id)
	return err
}
