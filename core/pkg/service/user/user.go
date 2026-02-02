// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package user provides the user management for the Synnax cluster. user includes a
// service for managing users, a writer for creating and updating users, and a reader
// for reading users. Authentication and authorization of users is done separately in
// package auth.
package user

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
	"github.com/vmihailenco/msgpack/v5"
)

var _ gorp.Entry[uuid.UUID] = User{}

// GorpKey implements gorp.Entry.
func (u User) GorpKey() uuid.UUID { return u.Key }

// SetOptions implements gorp.Entry.
func (u User) SetOptions() []any { return nil }

// DecodeMsgpack implements msgpack.CustomDecoder, supporting both legacy "node_id"
// and new "leaseholder" field names for backward compatibility.
func (u *User) DecodeMsgpack(dec *msgpack.Decoder) error {
	type alias User
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if err = msgpack.Unmarshal(raw, (*alias)(u)); err != nil {
		return err
	}
	keyIsNil := u.Key == uuid.Nil
	usernameEmpty := len(u.Username) == 0
	if keyIsNil || usernameEmpty {
		var legacy struct {
			Username string
			Key      Key
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return err
		}
		if keyIsNil {
			u.Key = legacy.Key
		}
		if usernameEmpty {
			u.Username = legacy.Username
		}
	}
	return nil
}
