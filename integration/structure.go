// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package main

import (
	"encoding/json"
	"fmt"

	"github.com/synnaxlabs/x/errors"
)

type Operation string

const (
	Read   Operation = "read"
	Write  Operation = "write"
	Delete Operation = "delete"
	Stream Operation = "stream"
)

type NodeParams interface {
	serialize() []string
	ToPythonCommand(identifier string) string
	ToTSCommand(identifier string) string
}

type TestNode struct {
	Op     Operation  `json:"op"`
	Client string     `json:"client"`
	Delay  float32    `json:"delay"`
	Params NodeParams `json:"params"`
}

func (tn *TestNode) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if err := json.Unmarshal(raw["op"], &tn.Op); err != nil {
		return errors.Wrap(err, "op field not set")
	}
	if err := json.Unmarshal(raw["client"], &tn.Client); err != nil {
		return errors.Wrap(err, "client field not set")
	}
	if delayField, ok := raw["delay"]; ok {
		if err := json.Unmarshal(delayField, &tn.Delay); err != nil {
			return errors.Wrap(err, "delay field not set")
		}
	} else {
		tn.Delay = 0
	}

	switch tn.Op {
	case Write:
		var params WriteParams
		if err := json.Unmarshal(raw["params"], &params); err != nil {
			return err
		}
		tn.Params = params
		break
	case Read:
		var params ReadParams
		if err := json.Unmarshal(raw["params"], &params); err != nil {
			return err
		}
		tn.Params = params
		break
	case Stream:
		var params StreamParams
		if err := json.Unmarshal(raw["params"], &params); err != nil {
			return err
		}
		tn.Params = params
		break
	case Delete:
		var params DeleteParams
		if err := json.Unmarshal(raw["params"], &params); err != nil {
			return err
		}
		tn.Params = params
		break
	default:
		return fmt.Errorf("unknown operation: %s", tn.Op)
	}

	return nil
}

type TestStep = []TestNode

type TestSequence struct {
	Cluster ClusterParam `json:"cluster"`
	Setup   SetUpParam   `json:"setup"`
	Steps   []TestStep   `json:"steps"`
	Cleanup CleanUpParam `json:"cleanup"`
}

func UnmarshalJSON(b []byte) (seq TestSequence, err error) {
	err = json.Unmarshal(b, &seq)
	return
}
