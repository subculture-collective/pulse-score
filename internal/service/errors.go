package service

import "fmt"

// ValidationError indicates invalid input.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s: %s", e.Field, e.Message)
}

// ConflictError indicates a resource conflict (e.g., duplicate email).
type ConflictError struct {
	Resource string `json:"resource"`
	Message  string `json:"message"`
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("conflict: %s: %s", e.Resource, e.Message)
}

// AuthError indicates an authentication failure.
type AuthError struct {
	Message string `json:"message"`
}

func (e *AuthError) Error() string {
	return e.Message
}

// NotFoundError indicates a resource was not found.
type NotFoundError struct {
	Resource string `json:"resource"`
	Message  string `json:"message"`
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("not found: %s: %s", e.Resource, e.Message)
}

// ForbiddenError indicates insufficient permissions.
type ForbiddenError struct {
	Message string `json:"message"`
}

func (e *ForbiddenError) Error() string {
	return e.Message
}
