// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package user_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("DecodeMsgpack", func() {
	It("Should decode new lowercase msgpack fields", func() {
		original := user.User{
			Key:       uuid.New(),
			Username:  "alice",
			FirstName: "Alice",
			LastName:  "Smith",
			RootUser:  true,
		}
		data := MustSucceed(msgpack.Marshal(original))
		var decoded user.User
		Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
		Expect(decoded.Key).To(Equal(original.Key))
		Expect(decoded.Username).To(Equal("alice"))
		Expect(decoded.FirstName).To(Equal("Alice"))
		Expect(decoded.LastName).To(Equal("Smith"))
		Expect(decoded.RootUser).To(BeTrue())
	})

	It("Should decode legacy uppercase msgpack fields", func() {
		key := uuid.New()
		legacy := struct {
			Key      uuid.UUID
			Username string
		}{
			Key:      key,
			Username: "bob",
		}
		data := MustSucceed(msgpack.Marshal(legacy))
		var decoded user.User
		Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
		Expect(decoded.Key).To(Equal(key))
		Expect(decoded.Username).To(Equal("bob"))
	})

	It("Should decode mixed legacy and new fields", func() {
		key := uuid.New()
		mixed := map[string]any{
			"Key":        key,
			"username":   "charlie",
			"first_name": "Charlie",
			"last_name":  "Brown",
		}
		data := MustSucceed(msgpack.Marshal(mixed))
		var decoded user.User
		Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
		Expect(decoded.Key).To(Equal(key))
		Expect(decoded.Username).To(Equal("charlie"))
		Expect(decoded.FirstName).To(Equal("Charlie"))
		Expect(decoded.LastName).To(Equal("Brown"))
	})
})
