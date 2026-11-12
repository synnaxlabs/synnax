// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

const notFileName = "file_name: must be a file name, not a path"

var _ = Describe("Writer", func() {
	var (
		e  *env
		rk resource.ID
	)
	BeforeEach(func(ctx SpecContext) {
		e = newEnv(ctx)
		rk = rack.Key(1).ResourceID()
	})

	fileSecret := func(fileName string) secret.Secret {
		return secret.Secret{Variant: secret.FileSecret{
			Base:     secret.Base{Name: "A"},
			FileName: fileName,
		}}
	}

	resolve := func(ctx SpecContext, key secret.Key) string {
		GinkgoHelper()
		g := MustSucceed(e.svc.Grant(ctx, e.db, rk, key))
		return string(MustSucceed(e.svc.Resolve(ctx, rk, g, key))[key])
	}

	Describe("Set", func() {
		It(
			"Should give a stored secret a key and encrypt its value",
			func(ctx SpecContext) {
				s := e.createStored(ctx, "Password", "hunter2")
				Expect(s.Base().Key).ToNot(Equal(uuid.Nil()))
				Expect(s.Base().Updated).To(Equal(e.now))
				Expect(resolve(ctx, s.Base().Key)).To(Equal("hunter2"))
			},
		)
		It("Should keep a key the caller gives", func(ctx SpecContext) {
			key := uuid.New()
			s := secret.Secret{Variant: secret.StoredSecret{
				Base: secret.Base{Key: key, Name: "Password"},
			}}
			Expect(e.write(ctx, func(w secret.Writer) error {
				return w.Set(ctx, &s, []byte("a"))
			})).To(Succeed())
			Expect(e.retrieve(ctx, key).Base().Name).To(Equal("Password"))
		})
		It("Should save the file name of a file secret", func(ctx SpecContext) {
			s := secret.Secret{Variant: secret.FileSecret{
				Base:     secret.Base{Name: "Client key"},
				FileName: "key.pem",
			}}
			Expect(e.write(ctx, func(w secret.Writer) error {
				return w.Set(ctx, &s, nil)
			})).To(Succeed())
			file := e.retrieve(ctx, s.Base().Key).Variant.(secret.FileSecret)
			Expect(file.FileName).To(Equal("key.pem"))
		})
		It(
			"Should replace the value of the secret with the same key",
			func(ctx SpecContext) {
				s := e.createStored(ctx, "Password", "old")
				e.now += 10
				Expect(e.write(ctx, func(w secret.Writer) error {
					return w.Set(ctx, &s, []byte("new"))
				})).To(Succeed())
				Expect(e.retrieve(ctx, s.Base().Key).Base().Updated).To(Equal(e.now))
				Expect(resolve(ctx, s.Base().Key)).To(Equal("new"))
			},
		)
		DescribeTable("Should refuse a replace that changes the source",
			func(ctx SpecContext, prev, next secret.Secret, value, msg string) {
				prevValue := []byte("hunter2")
				if prev.Source() == secret.SourceFile {
					prevValue = nil
				}
				Expect(e.write(ctx, func(w secret.Writer) error {
					return w.Set(ctx, &prev, prevValue)
				})).To(Succeed())
				base := next.Base()
				base.Key = prev.Base().Key
				next.SetBase(base)
				Expect(e.write(ctx, func(w secret.Writer) error {
					return w.Set(ctx, &next, []byte(value))
				})).To(MatchError(msg))
				Expect(e.retrieve(ctx, prev.Base().Key).Source()).
					To(Equal(prev.Source()))
			},
			Entry(
				"stored to file",
				secret.Secret{
					Variant: secret.StoredSecret{Base: secret.Base{Name: "A"}},
				},
				fileSecret("key.pem"),
				"",
				"source: cannot change from stored to file",
			),
			Entry(
				"file to stored",
				fileSecret("key.pem"),
				secret.Secret{
					Variant: secret.StoredSecret{Base: secret.Base{Name: "A"}},
				},
				"hunter2",
				"source: cannot change from file to stored",
			),
		)
		DescribeTable("Should reject an invalid secret",
			func(ctx SpecContext, s secret.Secret, value, msg string) {
				Expect(e.write(ctx, func(w secret.Writer) error {
					return w.Set(ctx, &s, []byte(value))
				})).
					To(MatchError(ContainSubstring(msg)))
			},
			Entry(
				"a stored secret with no value",
				secret.Secret{
					Variant: secret.StoredSecret{Base: secret.Base{Name: "A"}},
				},
				"",
				"value: required",
			),
			Entry(
				"a file secret with no file name",
				secret.Secret{Variant: secret.FileSecret{Base: secret.Base{Name: "A"}}},
				"",
				"file_name: required",
			),
			Entry(
				"a file secret with an absolute path",
				fileSecret("/etc/synnax/secrets.key"),
				"",
				"file_name: must be a file name, not a path",
			),
			Entry(
				"a file secret with a relative path",
				fileSecret("../secrets.key"),
				"",
				"file_name: must be a file name, not a path",
			),
			Entry(
				"a file secret with a Windows path",
				fileSecret(`C:\synnax\secrets.key`),
				"",
				"file_name: must be a file name, not a path",
			),
			Entry(
				"a file secret with a Windows drive",
				fileSecret("C:secrets.key"),
				"",
				"file_name: must be a file name, not a path",
			),
			Entry(
				"a file secret that names the parent folder",
				fileSecret(".."),
				"",
				"file_name: must be a file name, not a path",
			),
			Entry(
				"a file secret with a value",
				fileSecret("a"),
				"v",
				"value: must be empty for a file secret",
			),
			Entry(
				"a secret with no name",
				secret.Secret{Variant: secret.StoredSecret{}},
				"v",
				"name: required",
			),
			Entry("a secret with no source", secret.Secret{}, "v", "source: required"),
		)
	})

	Describe("Rename", func() {
		It("Should rename the secret and keep its value", func(ctx SpecContext) {
			s := e.createStored(ctx, "Before", "hunter2")
			Expect(e.write(ctx, func(w secret.Writer) error {
				return w.Rename(ctx, s.Base().Key, "After")
			})).To(Succeed())
			Expect(e.retrieve(ctx, s.Base().Key).Base().Name).To(Equal("After"))
			Expect(resolve(ctx, s.Base().Key)).To(Equal("hunter2"))
		})
		It("Should reject an empty name", func(ctx SpecContext) {
			s := e.createStored(ctx, "Before", "hunter2")
			Expect(e.write(ctx, func(w secret.Writer) error {
				return w.Rename(ctx, s.Base().Key, "")
			})).
				To(MatchError(ContainSubstring("name: required")))
		})
		It("Should return not found for a missing secret", func(ctx SpecContext) {
			Expect(e.write(ctx, func(w secret.Writer) error {
				return w.Rename(ctx, uuid.New(), "A")
			})).
				To(MatchError(query.ErrNotFound))
		})
	})

	DescribeTable("ValidateFileName",
		func(name string, matcher types.GomegaMatcher) {
			v := validate.New("secret")
			secret.ValidateFileName(v, "file_name", name)
			Expect(v.Error()).To(matcher)
		},
		Entry("a file name", "key.pem", Succeed()),
		Entry("an empty name", "", MatchError("file_name: required")),
		Entry("the folder itself", ".", MatchError(notFileName)),
		Entry("the parent folder", "..", MatchError(notFileName)),
		Entry("a forward slash", "a/key.pem", MatchError(notFileName)),
		Entry("a backslash", `a\key.pem`, MatchError(notFileName)),
		Entry("a Windows drive", "C:key.pem", MatchError(notFileName)),
	)

	Describe("Delete", func() {
		It("Should delete the secret", func(ctx SpecContext) {
			s := e.createStored(ctx, "Password", "hunter2")
			Expect(e.write(ctx, func(w secret.Writer) error {
				return w.Delete(ctx, s.Base().Key)
			})).To(Succeed())
			Expect(e.svc.NewRetrieve().
				Where(secret.MatchKeys(s.Base().Key)).
				Exec(ctx, e.db)).To(MatchError(query.ErrNotFound))
		})
		It("Should succeed for a missing secret", func(ctx SpecContext) {
			Expect(e.write(ctx, func(w secret.Writer) error {
				return w.Delete(ctx, uuid.New())
			})).To(Succeed())
		})
	})
})
