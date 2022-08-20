package fmock

import "github.com/arya-analytics/freighter"

var reporter = freighter.Reporter{
	Protocol:  "golang-mock",
	Encodings: []string{"in-memory"},
}
