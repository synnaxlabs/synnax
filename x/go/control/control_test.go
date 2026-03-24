// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package control_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/control"
)

var _ = Describe("State", func() {
	Describe("Subject", func() {
		Describe("String", func() {
			It("Should return both the key and name when present", func() {
				s := control.Subject{Key: "cat", Name: "Hat"}
				Expect(s.String()).To(Equal("[Hat]<cat>"))
			})
			It("Should return just the key when the name is absent", func() {
				s := control.Subject{Key: "cat"}
				Expect(s.String()).To(Equal("<cat>"))
			})
		})
		Describe("Validate", func() {
			It("Should return no error when key is set", func() {
				s := control.Subject{Key: "mykey", Name: "myname"}
				Expect(s.Validate()).To(Succeed())
			})
			It("Should return an error when key is empty", func() {
				s := control.Subject{Name: "myname"}
				Expect(s.Validate()).To(HaveOccurred())
			})
		})
		Describe("Override", func() {
			It("Should override zero-valued fields from other", func() {
				s := control.Subject{Key: "mykey"}
				other := control.Subject{Name: "myname", Group: 42}
				result := s.Override(other)
				Expect(result.Key).To(Equal("mykey"))
				Expect(result.Name).To(Equal("myname"))
				Expect(result.Group).To(Equal(uint32(42)))
			})
			It("Should override with non-zero values from other", func() {
				s := control.Subject{Key: "original", Name: "original", Group: 1}
				other := control.Subject{Key: "other", Name: "other", Group: 2}
				result := s.Override(other)
				Expect(result.Key).To(Equal("other"))
				Expect(result.Name).To(Equal("other"))
				Expect(result.Group).To(Equal(uint32(2)))
			})
			It("Should return the base subject when other is zero-valued", func() {
				s := control.Subject{Key: "mykey", Name: "myname", Group: 5}
				result := s.Override(control.Subject{})
				Expect(result).To(Equal(s))
			})
		})
	})

	Describe("State", func() {
		Describe("String", func() {
			It("Should return a nicely formatted state string", func() {
				s := control.Subject{Key: "cat", Name: "Hat"}
				state := control.State[int]{
					Subject:   s,
					Resource:  1,
					Authority: control.AuthorityAbsolute,
				}
				Expect(state.String()).To(Equal("[Hat]<cat> with authority 255 over 1"))
			})
		})
	})

	Describe("Transfer", func() {
		state1 := control.State[string]{
			Subject:   control.Subject{Key: "cat", Name: "Hat"},
			Resource:  "cookie",
			Authority: 12,
		}
		state2 := control.State[string]{
			Subject:   control.Subject{Key: "shredr", Name: "Dog"},
			Resource:  "cookie",
			Authority: 13,
		}
		Describe("IsAcquire", func() {
			It("Should return false when from is not nil", func() {
				Expect(control.Transfer[string]{From: &state1, To: &state2}.IsAcquire()).To(BeFalse())
			})

			It("Should return false when both from and to are nil", func() {
				Expect(control.Transfer[string]{From: nil, To: nil}.IsAcquire()).To(BeFalse())
			})

			It("Should return true when from is nil and to is not nil", func() {
				Expect(control.Transfer[string]{From: nil, To: &state2}.IsAcquire()).To(BeTrue())
			})
		})

		Describe("IsRelease", func() {
			It("Should return false when to is not nil", func() {
				Expect(control.Transfer[string]{From: &state1, To: &state2}.IsRelease()).To(BeFalse())
			})

			It("Should return false when both from and to are nil", func() {
				Expect(control.Transfer[string]{From: nil, To: nil}.IsRelease()).To(BeFalse())
			})

			It("Should return true when from is not nil and to is nil", func() {
				Expect(control.Transfer[string]{From: &state1, To: nil}.IsRelease()).To(BeTrue())
			})

		})

		Describe("IsTransfer", func() {
			It("Should return false when from is not nil and to is nil", func() {
				Expect(control.Transfer[string]{From: &state1, To: nil}.IsTransfer()).To(BeFalse())
			})

			It("Should return false when both from and to is nil and is nil", func() {
				Expect(control.Transfer[string]{From: nil, To: nil}.IsTransfer()).To(BeFalse())
			})

			It("Should return false when from is nil and to is not nil", func() {
				Expect(control.Transfer[string]{From: nil, To: &state2}.IsTransfer()).To(BeFalse())
			})

			It("Should return true when both from and to are not nil", func() {
				Expect(control.Transfer[string]{From: &state1, To: &state2}.IsTransfer()).To(BeTrue())
			})

			It("Should return false when both from and to are the same", func() {
				Expect(control.Transfer[string]{From: &state2, To: &state2}.IsTransfer()).To(BeFalse())
			})
		})

		Describe("Occurred", func() {
			It("Should return false when both to and from are nil", func() {
				Expect(control.Transfer[string]{From: nil, To: nil}.Occurred()).To(BeFalse())
			})

			It("Should return false when both the from and to states are the same", func() {
				Expect(control.Transfer[string]{From: &state1, To: &state1}.Occurred()).To(BeFalse())
			})
		})

		Describe("String", func() {
			Context("Release", func() {
				It("Should return a nicely formatted release string", func() {
					releaseT := control.Transfer[string]{
						From: &state1,
						To:   nil,
					}
					Expect(releaseT.String()).To(Equal(
						"[Hat]<cat>(12) released cookie",
					))
				})
			})

			Context("Acquire", func() {
				It("Should return a nicely formatted release string", func() {
					acquireT := control.Transfer[string]{
						From: nil,
						To:   &state2,
					}
					Expect(acquireT.String()).To(Equal(
						"[Dog]<shredr>(13) acquired cookie",
					))
				})
			})

			Context("Transfer", func() {
				It("Should return a nicely formatted release string", func() {
					transferT := control.Transfer[string]{
						From: &state1,
						To:   &state2,
					}
					Expect(transferT.String()).To(Equal(
						"transfer over cookie from [Hat]<cat>(12) to [Dog]<shredr>(13)",
					))
				})
			})

			Context("No Transfer", func() {
				It("Should return a nicely formatted string", func() {
					Expect(control.Transfer[string]{}.String()).To(Equal("no transfer occurred"))
				})
			})
		})
	})

})
