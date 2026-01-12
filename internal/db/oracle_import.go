//go:build oracle
// +build oracle

package db

import (
	_ "github.com/godror/godror" // only imported when building with -tags oracle
)
