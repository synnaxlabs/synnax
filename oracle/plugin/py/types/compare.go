// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/synnaxlabs/oracle/domain/validation"
	"github.com/synnaxlabs/oracle/resolution"
)

// lessThanValidators returns the model validators of the lt rules on fields. A class
// inherits the validators of its base classes, so fields holds only the fields that
// the class declares.
func lessThanValidators(fields []resolution.Field, data *templateData) string {
	var validators []string
	for _, f := range fields {
		lt := validation.Parse(f.Domains["validate"]).LessThan
		if lt == nil {
			continue
		}
		message := lt.Message
		if message == "" {
			message = "must be less than " + lt.Field
		}
		name, other := fieldName(f.Name), fieldName(lt.Field)
		validators = append(validators, fmt.Sprintf(`    @model_validator(mode="after")
    def _check_%s(self) -> Self:
        if not self.%s < self.%s:
            raise ValueError(%s)
        return self`, name, name, other, strconv.Quote(name+": "+message)))
	}
	if len(validators) > 0 {
		data.imports.addPydantic("model_validator")
		data.imports.addTyping("Self")
	}
	return strings.Join(validators, "\n\n")
}
