// Copyright 2019 Einride AB. Modified by Synnax Labs, Inc.
//
// Use of this source code is governed by the MIT license in the LICENSE file in this
// directory.

package dbc

import "github.com/synnaxlabs/x/errors"

// Keyword is a DBC keyword.
type Keyword string

const (
	KeywordAttribute               Keyword = "BA_DEF_"
	KeywordAttributeDefault        Keyword = "BA_DEF_DEF_"
	KeywordAttributeValue          Keyword = "BA_"
	KeywordBitTiming               Keyword = "BS_"
	KeywordComment                 Keyword = "CM_"
	KeywordEnvironmentVariable     Keyword = "EV_"
	KeywordEnvironmentVariableData Keyword = "ENVVAR_DATA_"
	KeywordMessage                 Keyword = "BO_"
	KeywordMessageTransmitters     Keyword = "BO_TX_BU_"
	KeywordMultiplexedValues       Keyword = "SG_MUL_VAL_"
	KeywordNewSymbols              Keyword = "NS_"
	KeywordNodes                   Keyword = "BU_"
	KeywordSignal                  Keyword = "SG_"
	KeywordSignalValueType         Keyword = "SIG_VALTYPE_"
	KeywordValueDescriptions       Keyword = "VAL_"
	KeywordValueTable              Keyword = "VAL_TABLE_"
	KeywordVersion                 Keyword = "VERSION"
)

// Identifier is a DBC identifier.
type Identifier string

// maxIdentifierLength is the length of the longest valid DBC identifier.
const maxIdentifierLength = 128

func isAlpha(r rune) bool { return ('A' <= r && r <= 'Z') || ('a' <= r && r <= 'z') }

func isDigit(r rune) bool { return '0' <= r && r <= '9' }

func (id Identifier) validate() error {
	if len(id) == 0 {
		return errors.Newf("invalid identifier '%s': zero-length", id)
	}
	if len(id) > maxIdentifierLength {
		return errors.Newf(
			"invalid identifier '%s': length %v exceeds max length %v",
			id,
			len(id),
			maxIdentifierLength,
		)
	}
	for i, r := range id {
		if i == 0 && r != '_' && !isAlpha(r) {
			return errors.Newf(
				"invalid identifier '%s': invalid first char: '%v'",
				id,
				r,
			)
		}
		if i > 0 && r != '_' && !isAlpha(r) && !isDigit(r) {
			return errors.Newf("invalid identifier '%s': invalid char: '%v'", id, r)
		}
	}
	return nil
}

// MessageID is the ID of a message. The most significant bit marks an extended CAN ID.
type MessageID uint32

const (
	maxID         = 0x7ff
	maxExtendedID = 0x1fffffff
	// messageIDExtendedFlag is set for extended message IDs.
	messageIDExtendedFlag MessageID = 0x80000000
	// messageIDIndependentSignals is the ID of the message that holds the signals no
	// message carries.
	messageIDIndependentSignals MessageID = 0xc0000000
)

// IsExtended reports whether the message ID is an extended CAN ID.
func (m MessageID) IsExtended() bool {
	return m != messageIDIndependentSignals && m&messageIDExtendedFlag > 0
}

// ToCAN returns the CAN ID of the message ID, without the extended flag.
func (m MessageID) ToCAN() uint32 {
	return uint32(m &^ messageIDExtendedFlag)
}

func (m MessageID) validate() error {
	if m == messageIDIndependentSignals {
		return nil
	}
	if m.IsExtended() && m.ToCAN() > maxExtendedID {
		return errors.Newf("invalid extended ID: %v", m)
	}
	if !m.IsExtended() && m.ToCAN() > maxID {
		return errors.Newf("invalid standard ID: %v", m)
	}
	return nil
}

// ObjectType is the type of object a definition applies to.
type ObjectType string

const (
	ObjectTypeUnspecified         ObjectType = ""
	ObjectTypeNetworkNode         ObjectType = "BU_"
	ObjectTypeMessage             ObjectType = "BO_"
	ObjectTypeSignal              ObjectType = "SG_"
	ObjectTypeEnvironmentVariable ObjectType = "EV_"
)

func (o ObjectType) validate() error {
	switch o {
	case ObjectTypeUnspecified,
		ObjectTypeNetworkNode,
		ObjectTypeMessage,
		ObjectTypeSignal,
		ObjectTypeEnvironmentVariable:
		return nil
	}
	return errors.Newf("invalid object type: %v", o)
}

// AttributeValueType is the type of the values of an attribute.
type AttributeValueType string

const (
	AttributeValueTypeInt    AttributeValueType = "INT"
	AttributeValueTypeHex    AttributeValueType = "HEX"
	AttributeValueTypeFloat  AttributeValueType = "FLOAT"
	AttributeValueTypeString AttributeValueType = "STRING"
	AttributeValueTypeEnum   AttributeValueType = "ENUM"
)

func (a AttributeValueType) validate() error {
	switch a {
	case AttributeValueTypeInt,
		AttributeValueTypeHex,
		AttributeValueTypeFloat,
		AttributeValueTypeString,
		AttributeValueTypeEnum:
		return nil
	}
	return errors.Newf("invalid attribute value type: %v", a)
}

// SignalValueType is the extended value type of a signal.
type SignalValueType uint64

const (
	SignalValueTypeInt     SignalValueType = 0
	SignalValueTypeFloat32 SignalValueType = 1
	SignalValueTypeFloat64 SignalValueType = 2
)

func (s SignalValueType) validate() error {
	switch s {
	case SignalValueTypeInt, SignalValueTypeFloat32, SignalValueTypeFloat64:
		return nil
	}
	return errors.Newf("invalid signal value type: %v", s)
}

// EnvironmentVariableType is the type of an environment variable.
type EnvironmentVariableType uint64

const (
	EnvironmentVariableTypeInteger EnvironmentVariableType = 0
	EnvironmentVariableTypeFloat   EnvironmentVariableType = 1
	EnvironmentVariableTypeString  EnvironmentVariableType = 2
)

func (e EnvironmentVariableType) validate() error {
	switch e {
	case EnvironmentVariableTypeInteger,
		EnvironmentVariableTypeFloat,
		EnvironmentVariableTypeString:
		return nil
	}
	return errors.Newf("invalid environment variable type: %v", e)
}

// AccessType is the access type of an environment variable.
type AccessType string

const (
	AccessTypeUnrestricted AccessType = "DUMMY_NODE_VECTOR0"
	AccessTypeRead         AccessType = "DUMMY_NODE_VECTOR1"
	AccessTypeWrite        AccessType = "DUMMY_NODE_VECTOR2"
	AccessTypeReadWrite    AccessType = "DUMMY_NODE_VECTOR3"
)

func (a AccessType) validate() error {
	switch a {
	case AccessTypeUnrestricted, AccessTypeRead, AccessTypeWrite, AccessTypeReadWrite:
		return nil
	}
	return errors.Newf("invalid access type: %v", a)
}
