// Package templates embeds all go-tk template files into the binary.
package templates

import "embed"

//go:embed all:project all:crud
var FS embed.FS
