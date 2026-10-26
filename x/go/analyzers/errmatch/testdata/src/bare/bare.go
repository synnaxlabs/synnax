// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bare

func Specs() {
	var (
		err    error
		custom *customErr
	)
	Expect(err).To(HaveOccurred())                       // want "assertion only checks that an error occurred"
	Expect(err).Should(HaveOccurred())                   // want "assertion only checks that an error occurred"
	Ω(err).Should(HaveOccurred())                        // want "assertion only checks that an error occurred"
	Expect(err).To(HaveOccurred(), "with a description") // want "assertion only checks that an error occurred"
	Expect(pair()).Error().To(HaveOccurred())            // want "assertion only checks that an error occurred"
	Eventually(fails).Should(HaveOccurred())             // want "assertion only checks that an error occurred"
	Expect(err).ToNot(Not(HaveOccurred()))               // want "assertion only checks that an error occurred"
	Expect(fails()).ToNot(Succeed())                     // want "assertion only checks that an error occurred"
	Expect(fails()).NotTo(Succeed())                     // want "assertion only checks that an error occurred"
	Expect(fails()).ShouldNot(Succeed())                 // want "assertion only checks that an error occurred"
	Expect(fails()).To(Not(Succeed()))                   // want "assertion only checks that an error occurred"
	Expect(err).ToNot(BeNil())                           // want "assertion only checks that an error occurred"
	Expect(err).To(Not(BeNil()))                         // want "assertion only checks that an error occurred"
	Expect(custom).ToNot(BeNil())                        // want "assertion only checks that an error occurred"
	ExpectWithOffset(1, err).ToNot(BeNil())              // want "assertion only checks that an error occurred"
	Expect(pair()).Error().ToNot(BeNil())                // want "assertion only checks that an error occurred"
}
