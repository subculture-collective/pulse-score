package auth

import (
	"context"

	"github.com/google/uuid"
)

type contextKey int

const (
	userIDKey contextKey = iota
	orgIDKey
	roleKey
)

// WithUserID sets the user ID in context.
func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// WithOrgID sets the organization ID in context.
func WithOrgID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, orgIDKey, id)
}

// WithRole sets the user's role in context.
func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, roleKey, role)
}

// GetUserID extracts the user ID from context.
func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}

// GetOrgID extracts the organization ID from context.
func GetOrgID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(orgIDKey).(uuid.UUID)
	return id, ok
}

// GetRole extracts the user's role from context.
func GetRole(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(roleKey).(string)
	return role, ok
}
