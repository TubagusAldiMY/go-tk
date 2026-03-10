package new

import "errors"

// ErrPromptCancelled is returned when the user cancels the interactive prompts.
var ErrPromptCancelled = errors.New("prompt cancelled by user")

// ErrProjectExists is returned when the target directory already exists and is non-empty.
var ErrProjectExists = errors.New("target directory already exists")
