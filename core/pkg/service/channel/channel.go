// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import "github.com/synnaxlabs/synnax/pkg/distribution/channel"

type (
	Key           = channel.Key
	Keys          = channel.Keys
	LocalKey      = channel.LocalKey
	Channel       = channel.Channel
	Operation     = channel.Operation
	OperationType = channel.OperationType
	CreateOption  = channel.CreateOption
	Retrieve      = channel.Retrieve
)

const (
	OperationTypeMin        = channel.OperationTypeMin
	OperationTypeMax        = channel.OperationTypeMax
	OperationTypeAvg        = channel.OperationTypeAvg
	OperationTypeNone       = channel.OperationTypeNone
	OperationTypeDerivative = channel.OperationTypeDerivative
)

var (
	RetrieveIfNameExists                        = channel.RetrieveIfNameExists
	OverwriteIfNameExistsAndDifferentProperties = channel.OverwriteIfNameExistsAndDifferentProperties
	CreateWithoutGroupRelationship              = channel.CreateWithoutGroupRelationship
	ParseKey                                    = channel.ParseKey
	OntologyID                                  = channel.OntologyID
	MatchKeys                                   = channel.MatchKeys
	MatchNames                                  = channel.MatchNames
	OntologyIDsFromChannels                     = channel.OntologyIDsFromChannels
	KeysFromChannels                            = channel.KeysFromChannels
	MatchLeaseholders                           = channel.MatchLeaseholders
	MatchDataTypes                              = channel.MatchDataTypes
	MatchVirtual                                = channel.MatchVirtual
	MatchIsIndex                                = channel.MatchIsIndex
	MatchInternal                               = channel.MatchInternal
	ToPayload                                   = channel.ToPayload
	MatchCalculated                             = channel.MatchCalculated
	Not                                         = channel.Not
	NewRandomName                               = channel.NewRandomName
)
