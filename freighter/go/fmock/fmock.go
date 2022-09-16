package fmock

import "github.com/synnaxlabs/freighter"

var reporter = freighter.Reporter{
	Protocol:  "golang-mock",
	Encodings: []string{"in-memory"},
}
