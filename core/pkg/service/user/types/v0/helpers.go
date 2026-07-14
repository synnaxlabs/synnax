// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/gorp"
	"github.com/vmihailenco/msgpack/v5"
)

var _ gorp.Entry[Key] = User{}

// OntologyID returns a unique identifier for the user for use within a resource
// ontology.
func (u User) OntologyID() ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeUser, Key: u.Key.String()}
}

// GorpKey implements gorp.Entry.
func (u User) GorpKey() Key { return u.Key }

// SetOptions implements gorp.Entry.
func (u User) SetOptions() []any { return nil }

// DecodeMsgpack implements msgpack.CustomDecoder, supporting both legacy uppercase
// msgpack field names (e.g. "Key", "Username") and new lowercase field names for
// backward compatibility.
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
