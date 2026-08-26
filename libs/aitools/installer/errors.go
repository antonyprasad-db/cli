package installer

import "errors"

// Sentinel errors that let the command layer categorize an install failure
// without matching on error-message strings. Wrap them with %w at the site
// that produces the failure; classify with errors.Is upstream.
var (
	// ErrSkillNotFound is returned when a skill named via --skills is absent from
	// the resolved manifest.
	ErrSkillNotFound = errors.New("skill not found")

	// ErrVersionIncompatible is returned when a skill requires a newer CLI than
	// the one running.
	ErrVersionIncompatible = errors.New("skill requires a newer CLI version")
)
