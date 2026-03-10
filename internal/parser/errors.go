package parser

import "errors"

// ErrStructNotFound is returned when a named struct cannot be found in a file.
var ErrStructNotFound = errors.New("struct not found")

// ErrRouteParseFailure is returned when route extraction fails completely.
var ErrRouteParseFailure = errors.New("route parse failure")
