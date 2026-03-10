package crud

import "errors"

// ErrNoConfig is returned when gotk.yaml is not found in the current project.
var ErrNoConfig = errors.New("gotk.yaml not found — run this command from your project root")

// ErrEntityExists is returned when the entity files already exist and --force is not set.
var ErrEntityExists = errors.New("entity files already exist; use --force to overwrite")
