// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package good

func Specs() {
	var err error
	Expect(err).To(MatchError(errSentinel))
	Expect(err).To(MatchError(ContainSubstring("not found")))
	Expect(pair()).Error().To(MatchError(errSentinel))
	Expect(err).ToNot(HaveOccurred())
	Expect(err).To(Not(HaveOccurred()))
	Expect(fails()).To(Succeed())
	Expect(fails()).ToNot(Not(Succeed()))
	Expect(err).To(BeNil())
	Expect(ptr()).ToNot(BeNil())
	Expect(pair()).ToNot(BeNil())
	Eventually(ptr).ShouldNot(BeNil())
	Eventually(ptr).WithTimeout(1).ShouldNot(BeNil())
	Eventually(pair).ShouldNot(BeNil())
	Expect(ptr(), err).ToNot(BeNil())
}
