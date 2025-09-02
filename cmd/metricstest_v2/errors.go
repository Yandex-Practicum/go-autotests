package main

import (
	"errors"
)

var (
	// errUsageFound indicates presence of some object
	errUsageFound = errors.New("usage found")
	// errUsageNotFound indicates absence of some object
	errUsageNotFound = errors.New("usage not found")
)
