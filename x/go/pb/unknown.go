// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb

import (
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// RejectUnknown returns an error wrapping validate.ErrValidation if msg or a message
// nested in it holds a field that its schema does not declare. The contents of a
// google.protobuf.Any are not checked.
func RejectUnknown(msg proto.Message) error {
	return rejectUnknown(msg.ProtoReflect())
}

func rejectUnknown(m protoreflect.Message) error {
	if len(m.GetUnknown()) > 0 {
		return errors.Wrapf(
			validate.ErrValidation,
			"%s has unknown fields",
			m.Descriptor().FullName(),
		)
	}
	var err error
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch {
		case fd.IsMap():
			if fd.MapValue().Message() == nil {
				return true
			}
			v.Map().Range(func(_ protoreflect.MapKey, e protoreflect.Value) bool {
				err = rejectUnknown(e.Message())
				return err == nil
			})
		case fd.IsList():
			if fd.Message() == nil {
				return true
			}
			for i := range v.List().Len() {
				if err = rejectUnknown(v.List().Get(i).Message()); err != nil {
					break
				}
			}
		case fd.Message() != nil:
			err = rejectUnknown(v.Message())
		}
		return err == nil
	})
	return err
}
