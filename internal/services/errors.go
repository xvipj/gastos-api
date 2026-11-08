// Package services contiene la lógica de negocio. No importa Gin ni conoce
// detalles de HTTP: recibe y devuelve datos, y expone errores tipados.
package services

import "errors"

// Errores centinela que los handlers traducen a códigos HTTP.
var (
	ErrValidation    = errors.New("validation error")
	ErrUnprocessable = errors.New("unprocessable entity")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrNotFound      = errors.New("not found")
	ErrConflict      = errors.New("conflict")
)

// ServiceError es un error de negocio con un mensaje seguro para el cliente.
type ServiceError struct {
	kind    error
	message string
}

// Error implementa la interfaz error.
func (e *ServiceError) Error() string { return e.message }

// Unwrap permite usar errors.Is con los errores centinela.
func (e *ServiceError) Unwrap() error { return e.kind }

// NewValidationError construye un error de entrada mal formada (HTTP 400).
func NewValidationError(message string) error {
	return &ServiceError{kind: ErrValidation, message: message}
}

// NewUnprocessableError construye un error de regla de negocio (HTTP 422).
func NewUnprocessableError(message string) error {
	return &ServiceError{kind: ErrUnprocessable, message: message}
}

// NewUnauthorizedError construye un error de autenticación (HTTP 401).
func NewUnauthorizedError(message string) error {
	return &ServiceError{kind: ErrUnauthorized, message: message}
}

// NewNotFoundError construye un error de recurso inexistente (HTTP 404).
func NewNotFoundError(message string) error {
	return &ServiceError{kind: ErrNotFound, message: message}
}

// NewConflictError construye un error de conflicto (HTTP 409).
func NewConflictError(message string) error {
	return &ServiceError{kind: ErrConflict, message: message}
}
