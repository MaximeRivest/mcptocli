package exitcode

import (
	"errors"
	"fmt"
)

// Category identifies a stable class of CLI failure.
type Category int

const (
	Internal Category = iota
	Usage
	Config
	Auth
	Transport
	Protocol
	Server
	Interactive
)

// Error is a categorized CLI error with an optional hint.
type Error struct {
	Category Category
	Message  string
	Hint     string
	Err      error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "unknown error"
}

func (e *Error) Unwrap() error { return e.Err }

// New creates a categorized error.
func New(category Category, message string) error {
	return &Error{Category: category, Message: message}
}

// Newf creates a categorized error using fmt.Sprintf.
func Newf(category Category, format string, args ...any) error {
	return &Error{Category: category, Message: fmt.Sprintf(format, args...)}
}

// Wrap attaches category and message to an underlying error.
func Wrap(category Category, err error, message string) error {
	if err == nil {
		return nil
	}
	return &Error{Category: category, Message: message, Err: err}
}

// Wrapf attaches category and formatted message to an underlying error.
func Wrapf(category Category, err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return &Error{Category: category, Message: fmt.Sprintf(format, args...), Err: err}
}

// WithHint attaches a hint to an error while preserving its category.
func WithHint(err error, hint string) error {
	if err == nil || hint == "" {
		return err
	}
	var cliErr *Error
	if errors.As(err, &cliErr) {
		clone := *cliErr
		clone.Hint = hint
		return &clone
	}
	return &Error{Category: Internal, Message: err.Error(), Hint: hint, Err: err}
}

// Code maps an error to a stable process exit code.
func Code(err error) int {
	if err == nil {
		return 0
	}
	var cliErr *Error
	if !errors.As(err, &cliErr) {
		return 10
	}

	switch cliErr.Category {
	case Usage:
		return 2
	case Config:
		return 3
	case Auth:
		return 4
	case Transport:
		return 5
	case Protocol:
		return 6
	case Server:
		return 7
	case Interactive:
		return 8
	default:
		return 10
	}
}

// Format renders an error for stderr output.
func Format(err error) string {
	if err == nil {
		return ""
	}
	var cliErr *Error
	if errors.As(err, &cliErr) {
		if cliErr.Hint != "" {
			return fmt.Sprintf("error: %s\nhint: %s", cliErr.Error(), cliErr.Hint)
		}
		return fmt.Sprintf("error: %s", cliErr.Error())
	}
	return fmt.Sprintf("error: %s", err)
}
