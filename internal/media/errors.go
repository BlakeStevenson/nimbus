package media

import "errors"

var (
	// ErrNotFound is returned when a media item is not found
	ErrNotFound = errors.New("media item not found")

	// ErrInvalidKind is returned when an invalid media kind is provided
	ErrInvalidKind = errors.New("invalid media kind")

	// ErrTitleRequired is returned when title is empty
	ErrTitleRequired = errors.New("title is required")

	// ErrInvalidFilter is returned when filter parameters are invalid
	ErrInvalidFilter = errors.New("invalid filter parameters")
)
