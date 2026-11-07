// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bad

func bad() {
	var db closer
	AfterEach(func() {
		Expect(db.Close()).To(Succeed()) // want "not in AfterEach"
	})
	JustAfterEach(func() {
		_ = db.Close() // want "not in JustAfterEach"
	})
	AfterAll(func() {
		for range 2 {
			Expect(db.Close()).To(Succeed()) // want "not in AfterAll"
		}
	})
	AfterSuite(func() {
		Expect(db.Close()).To(Succeed()) // want "not in AfterSuite"
	})
}
