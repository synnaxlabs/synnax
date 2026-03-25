// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrate

import (
	"fmt"
	"strings"

	"github.com/synnaxlabs/x/gorp"
)

// layoutToGo converts a []gorp.FieldLayout into a Go source code literal string
// that can be embedded in generated migration registration files.
func layoutToGo(layouts []gorp.FieldLayout, indent string) string {
	var b strings.Builder
	b.WriteString("[]gorp.FieldLayout{\n")
	for _, f := range layouts {
		fieldToGo(&b, f, indent+"\t")
	}
	b.WriteString(indent + "}")
	return b.String()
}

func fieldToGo(b *strings.Builder, f gorp.FieldLayout, indent string) {
	b.WriteString(indent + "{")
	b.WriteString(fmt.Sprintf("Name: %q, Encoding: %s", f.Name, encodingName(f.Encoding)))
	if f.Optional {
		b.WriteString(", Optional: true")
	}
	if f.HardOptional {
		b.WriteString(", HardOptional: true")
	}
	if len(f.Fields) > 0 {
		b.WriteString(", Fields: " + layoutToGo(f.Fields, indent+"\t"))
	}
	if f.Element != nil {
		b.WriteString(", Element: &gorp.FieldLayout{")
		b.WriteString(fmt.Sprintf("Encoding: %s", encodingName(f.Element.Encoding)))
		if len(f.Element.Fields) > 0 {
			b.WriteString(", Fields: " + layoutToGo(f.Element.Fields, indent+"\t\t"))
		}
		if f.Element.Element != nil {
			b.WriteString(fmt.Sprintf(", Element: &gorp.FieldLayout{Encoding: %s}", encodingName(f.Element.Element.Encoding)))
		}
		b.WriteString("}")
	}
	if f.Key != nil {
		b.WriteString(fmt.Sprintf(", Key: &gorp.FieldLayout{Encoding: %s}", encodingName(f.Key.Encoding)))
	}
	if f.Value != nil {
		b.WriteString(fmt.Sprintf(", Value: &gorp.FieldLayout{Encoding: %s", encodingName(f.Value.Encoding)))
		if len(f.Value.Fields) > 0 {
			b.WriteString(", Fields: " + layoutToGo(f.Value.Fields, indent+"\t\t"))
		}
		b.WriteString("}")
	}
	b.WriteString("},\n")
}

func encodingName(e gorp.Encoding) string {
	switch e {
	case gorp.EncodingBool:
		return "gorp.EncodingBool"
	case gorp.EncodingInt8:
		return "gorp.EncodingInt8"
	case gorp.EncodingInt16:
		return "gorp.EncodingInt16"
	case gorp.EncodingInt32:
		return "gorp.EncodingInt32"
	case gorp.EncodingInt64:
		return "gorp.EncodingInt64"
	case gorp.EncodingUint8:
		return "gorp.EncodingUint8"
	case gorp.EncodingUint16:
		return "gorp.EncodingUint16"
	case gorp.EncodingUint32:
		return "gorp.EncodingUint32"
	case gorp.EncodingUint64:
		return "gorp.EncodingUint64"
	case gorp.EncodingFloat32:
		return "gorp.EncodingFloat32"
	case gorp.EncodingFloat64:
		return "gorp.EncodingFloat64"
	case gorp.EncodingUUID:
		return "gorp.EncodingUUID"
	case gorp.EncodingString:
		return "gorp.EncodingString"
	case gorp.EncodingBytes:
		return "gorp.EncodingBytes"
	case gorp.EncodingJSON:
		return "gorp.EncodingJSON"
	case gorp.EncodingStruct:
		return "gorp.EncodingStruct"
	case gorp.EncodingArray:
		return "gorp.EncodingArray"
	case gorp.EncodingMap:
		return "gorp.EncodingMap"
	default:
		return fmt.Sprintf("gorp.Encoding(%d)", e)
	}
}
