// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bus

import (
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// Uses returns the ICD messages a read or write task config uses, with the fields of
// each. It returns an error when config does not decode as a bus task config.
func Uses(config msgpack.EncodedJSON) ([]icd.Use, error) {
	var cfg struct {
		Messages []struct {
			Message icd.EntryKey `json:"message"`
			Fields  []struct {
				Field icd.FieldKey `json:"field"`
			} `json:"fields"`
		} `json:"messages"`
	}
	if err := config.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	uses := make([]icd.Use, len(cfg.Messages))
	for i, m := range cfg.Messages {
		uses[i] = icd.Use{Message: m.Message}
		for _, f := range m.Fields {
			uses[i].Fields = append(uses[i].Fields, f.Field)
		}
	}
	return uses, nil
}
