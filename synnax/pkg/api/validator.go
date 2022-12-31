// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api

import (
	"github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"reflect"
	"strings"
)

func newValidator() *validator.Validate {
	v := validator.New()
	v.RegisterTagNameFunc(tagNameFunc)
	return v
}

var tagNames = []string{"json", "msgpack"}

func tagNameFunc(fld reflect.StructField) string {
	if fld.Anonymous {
		return errors.EmbeddedFieldTag
	}
	for _, tagName := range tagNames {
		if name, ok := getTagName(fld, tagName); ok {
			return name
		}
	}
	zap.S().Warnf("no tag name found for field %s", fld.Name)
	return strings.ToLower(fld.Name)
}

func getTagName(fld reflect.StructField, tagName string) (string, bool) {
	name := strings.SplitN(fld.Tag.Get(tagName), ",", 2)[0]
	return name, name != "-"
}
