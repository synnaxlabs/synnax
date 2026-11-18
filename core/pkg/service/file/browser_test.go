// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package file_test

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func names(l listing) []string {
	out := make([]string, len(l.Entries))
	for i, e := range l.Entries {
		out[i] = e.Name
	}
	return out
}

// root returns the root of the volume that holds path.
func root(
	path string,
) string {
	return filepath.VolumeName(path) + string(filepath.Separator)
}

func find(l listing, name string) entry {
	GinkgoHelper()
	for _, e := range l.Entries {
		if e.Name == name {
			return e
		}
	}
	Fail(fmt.Sprintf("no entry %s in %s", name, l.Path))
	return entry{}
}

func write(path string, data []byte) {
	GinkgoHelper()
	Expect(os.WriteFile(path, data, 0o600)).To(Succeed())
}

func details(der []byte, name string) *certificate.Details {
	sum1, sum256 := sha1.Sum(der), sha256.Sum256(der)
	return &certificate.Details{
		Subject:   "CN=" + name,
		Issuer:    "CN=" + name,
		NotBefore: telem.NewTimeStamp(notAfter.AddDate(-1, 0, 0)),
		NotAfter:  telem.NewTimeStamp(notAfter),
		Sha1:      hex.EncodeToString(sum1[:]),
		Sha256:    hex.EncodeToString(sum256[:]),
	}
}

var _ = Describe("List", func() {
	var (
		b   browser
		dir string
	)
	BeforeEach(func(ctx SpecContext) {
		dir = GinkgoT().TempDir()
		b = configure(ctx, "")
	})

	Describe("Top level", func() {
		It("Should list the roots and the home folder", func(ctx SpecContext) {
			home := MustSucceed(os.UserHomeDir())
			l := b.list(ctx, "")
			Expect(l.Path).To(BeEmpty())
			Expect(len(l.Entries)).To(BeNumerically(">=", 2))
			roots, last := l.Entries[:len(l.Entries)-1], l.Entries[len(l.Entries)-1]
			Expect(last).To(Equal(entry{
				Name: filepath.Base(home),
				Path: home,
				Kind: "directory",
			}))
			Expect(roots).To(ContainElement(entry{
				Name: root(dir),
				Path: root(dir),
				Kind: "directory",
			}))
			for _, r := range roots {
				Expect(
					r,
				).To(Equal(entry{Name: r.Path, Path: r.Path, Kind: "directory"}))
				Expect(filepath.Dir(r.Path)).To(Equal(r.Path))
			}
		})

		It(
			"Should list the top level for a command with no path",
			func(ctx SpecContext) {
				stat := b.exec(ctx, nil)
				Expect(stat.Variant).To(Equal(status.VariantSuccess))
				var l listing
				Expect(stat.Details.Data.Unmarshal(&l)).To(Succeed())
				Expect(l).To(Equal(b.list(ctx, "")))
			},
		)

		It("Should list a folder once when the secrets folder is the home folder",
			func(ctx SpecContext) {
				l := b.list(ctx, "")
				b = configure(ctx, MustSucceed(os.UserHomeDir()))
				Expect(b.list(ctx, "").Entries).To(Equal(l.Entries))
			})

		It("Should take a relative secrets folder from the working folder",
			func(ctx SpecContext) {
				Expect(os.Mkdir(filepath.Join(dir, "secrets"), 0o700)).To(Succeed())
				GinkgoT().Chdir(dir)
				b = configure(ctx, "secrets")
				Expect(
					b.list(ctx, "").SecretsDir,
				).To(Equal(filepath.Join(dir, "secrets")))
			})

		It(
			"Should list the secrets folder last when one is set",
			func(ctx SpecContext) {
				b = configure(ctx, dir)
				l := b.list(ctx, "")
				Expect(l.SecretsDir).To(Equal(dir))
				Expect(l.Entries[len(l.Entries)-1]).To(Equal(entry{
					Name: filepath.Base(dir),
					Path: dir,
					Kind: "directory",
				}))
			},
		)
	})

	Describe("Folder", func() {
		It(
			"Should list folders first, then files, each by name",
			func(ctx SpecContext) {
				Expect(os.Mkdir(filepath.Join(dir, "b"), 0o700)).To(Succeed())
				Expect(os.Mkdir(filepath.Join(dir, "a"), 0o700)).To(Succeed())
				write(filepath.Join(dir, "z.txt"), nil)
				write(filepath.Join(dir, "c.txt"), nil)
				l := b.list(ctx, dir)
				Expect(l.Path).To(Equal(dir))
				Expect(l.Parent).To(Equal(filepath.Dir(dir)))
				Expect(l.Secret).To(BeFalse())
				Expect(l.Truncated).To(BeFalse())
				Expect(names(l)).To(Equal([]string{"a", "b", "c.txt", "z.txt"}))
				Expect(find(l, "a").Kind).To(Equal("directory"))
				Expect(find(l, "c.txt").Kind).To(Equal("file"))
				Expect(find(l, "c.txt").Path).To(Equal(filepath.Join(dir, "c.txt")))
			},
		)

		It(
			"Should give the size and modified time of each entry",
			func(ctx SpecContext) {
				modified := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
				path := filepath.Join(dir, "notes.txt")
				write(path, []byte("hello"))
				Expect(os.Chtimes(path, modified, modified)).To(Succeed())
				Expect(os.Mkdir(filepath.Join(dir, "sub"), 0o700)).To(Succeed())
				l := b.list(ctx, dir)
				notes := find(l, "notes.txt")
				Expect(notes.Size).To(HaveValue(Equal(int64(5))))
				Expect(
					notes.Modified,
				).To(HaveValue(Equal(telem.NewTimeStamp(modified))))
				Expect(find(l, "sub").Size).To(HaveValue(BeZero()))
			},
		)

		It("Should list a path in its lexically normal form", func(ctx SpecContext) {
			Expect(os.Mkdir(filepath.Join(dir, "a"), 0o700)).To(Succeed())
			l := b.list(ctx, dir+"/a/../")
			Expect(l.Path).To(Equal(dir))
			Expect(find(l, "a").Path).To(Equal(filepath.Join(dir, "a")))
		})

		It("Should give no parent for a root", func(ctx SpecContext) {
			l := b.list(ctx, root(dir))
			Expect(l.Path).To(Equal(root(dir)))
			Expect(l.Parent).To(BeEmpty())
		})

		It("Should keep the first 1000 entries", func(ctx SpecContext) {
			for i := range 1001 {
				write(filepath.Join(dir, fmt.Sprintf("f%04d", i)), nil)
			}
			Expect(os.Mkdir(filepath.Join(dir, "zz"), 0o700)).To(Succeed())
			l := b.list(ctx, dir)
			Expect(l.Truncated).To(BeTrue())
			Expect(l.Entries).To(HaveLen(1000))
			Expect(l.Entries[0].Name).To(Equal("zz"))
			Expect(l.Entries[999].Name).To(Equal("f0998"))
		})
	})

	Describe("Answer size", func() {
		It("Should leave out the entries past the size of an answer",
			func(ctx SpecContext) {
				deep := filepath.Join(
					dir,
					strings.Repeat("a", 200),
					strings.Repeat("b", 200),
					strings.Repeat("c", 200),
				)
				Expect(os.MkdirAll(deep, 0o700)).To(Succeed())
				for i := range 1000 {
					name := fmt.Sprintf("%04d%s", i, strings.Repeat("f", 236))
					write(filepath.Join(deep, name), nil)
				}
				l := b.list(ctx, deep)
				Expect(l.Truncated).To(BeTrue())
				Expect(len(l.Entries)).To(BeNumerically("<", 1000))
				Expect(l.Entries[0].Name).To(HavePrefix("0000"))
			})
	})

	Describe("Links", func() {
		It("Should follow a link to a folder", func(ctx SpecContext) {
			Expect(os.Mkdir(filepath.Join(dir, "a"), 0o700)).To(Succeed())
			Expect(os.Symlink(filepath.Join(dir, "a"), filepath.Join(dir, "link"))).
				To(Succeed())
			Expect(find(b.list(ctx, dir), "link").Kind).To(Equal("directory"))
		})

		It("Should leave out a link that leads nowhere", func(ctx SpecContext) {
			Expect(
				os.Symlink(filepath.Join(dir, "missing"), filepath.Join(dir, "broken")),
			).
				To(Succeed())
			Expect(b.list(ctx, dir).Entries).To(BeEmpty())
		})
	})

	Describe("Certificates", func() {
		It("Should give the details of a PEM certificate", func(ctx SpecContext) {
			der, _ := newCertificate("Broker")
			write(filepath.Join(dir, "ca.PEM"), encodePEM("CERTIFICATE", der))
			Expect(find(b.list(ctx, dir), "ca.PEM").Certificate).
				To(Equal(details(der, "Broker")))
		})

		It("Should give the details of a DER certificate", func(ctx SpecContext) {
			der, _ := newCertificate("Client")
			write(filepath.Join(dir, "client.der"), der)
			Expect(find(b.list(ctx, dir), "client.der").Certificate).
				To(Equal(details(der, "Client")))
		})

		It("Should give only the name, kind, size, and time of a key file",
			func(ctx SpecContext) {
				_, key := newCertificate("Client")
				write(filepath.Join(dir, "client.key"), key)
				write(filepath.Join(dir, "client-key.pem"), key)
				l := b.list(ctx, dir)
				for _, name := range []string{"client.key", "client-key.pem"} {
					e := find(l, name)
					Expect(e.Certificate).To(BeNil())
					Expect(e.Size).To(HaveValue(Equal(int64(len(key)))))
				}
			})

		It("Should not read a file without a certificate extension",
			func(ctx SpecContext) {
				der, _ := newCertificate("Broker")
				write(filepath.Join(dir, "ca.txt"), encodePEM("CERTIFICATE", der))
				Expect(find(b.list(ctx, dir), "ca.txt").Certificate).To(BeNil())
			})

		It("Should not read a file larger than 1 MiB", func(ctx SpecContext) {
			der, _ := newCertificate("Broker")
			data := append(
				encodePEM("CERTIFICATE", der),
				bytes.Repeat([]byte("\n"), 1<<20)...,
			)
			write(filepath.Join(dir, "ca.pem"), data)
			Expect(find(b.list(ctx, dir), "ca.pem").Certificate).To(BeNil())
		})
	})

	Describe("Secrets folder", func() {
		var secrets string
		BeforeEach(func(ctx SpecContext) {
			secrets = filepath.Join(dir, "secrets")
			Expect(os.Mkdir(secrets, 0o700)).To(Succeed())
			der, _ := newCertificate("Broker")
			write(filepath.Join(secrets, "ca.pem"), encodePEM("CERTIFICATE", der))
			b = configure(ctx, secrets)
		})

		It(
			"Should give only the name, path, and kind of each entry",
			func(ctx SpecContext) {
				l := b.list(ctx, secrets)
				Expect(l.Secret).To(BeTrue())
				Expect(l.SecretsDir).To(Equal(secrets))
				Expect(l.Entries).To(Equal([]entry{{
					Name: "ca.pem",
					Path: filepath.Join(secrets, "ca.pem"),
					Kind: "file",
				}}))
			},
		)

		It(
			"Should mark a folder in the secrets folder as secret",
			func(ctx SpecContext) {
				Expect(os.Mkdir(filepath.Join(secrets, "sub"), 0o700)).To(Succeed())
				Expect(b.list(ctx, filepath.Join(secrets, "sub")).Secret).To(BeTrue())
			},
		)

		It("Should give only the name of the secrets folder and of a link into it",
			func(ctx SpecContext) {
				Expect(os.Symlink(
					filepath.Join(secrets, "ca.pem"),
					filepath.Join(dir, "ca.pem"),
				)).To(Succeed())
				l := b.list(ctx, dir)
				Expect(l.Secret).To(BeFalse())
				Expect(find(l, "secrets").Size).To(BeNil())
				Expect(find(l, "ca.pem")).To(Equal(entry{
					Name: "ca.pem",
					Path: filepath.Join(dir, "ca.pem"),
					Kind: "file",
				}))
			})

		It("Should mark the secrets folder as secret by a name in another case",
			func(ctx SpecContext) {
				upper := filepath.Join(dir, "SECRETS")
				if _, err := os.Stat(upper); err != nil {
					Skip("the file system is case-sensitive")
				}
				l := b.list(ctx, upper)
				Expect(l.Secret).To(BeTrue())
				Expect(find(l, "ca.pem").Certificate).To(BeNil())
			})

		It("Should mark a link to the secrets folder as secret", func(ctx SpecContext) {
			alias := filepath.Join(dir, "alias")
			Expect(os.Symlink(secrets, alias)).To(Succeed())
			l := b.list(ctx, alias)
			Expect(l.Secret).To(BeTrue())
			Expect(find(l, "ca.pem").Certificate).To(BeNil())
		})
	})

	Describe("Failures", func() {
		fail := func(ctx SpecContext, args msgpack.EncodedJSON, description string) {
			GinkgoHelper()
			stat := b.exec(ctx, args)
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Message).To(Equal("Failed to list files"))
			Expect(stat.Description).To(Equal(description))
			Expect(stat.Details.Data).To(BeEmpty())
		}

		It("Should fail for a relative path", func(ctx SpecContext) {
			fail(
				ctx,
				msgpack.EncodedJSON{"path": "etc"},
				"etc is not an absolute path.",
			)
		})

		It("Should fail for a path where nothing is", func(ctx SpecContext) {
			path := filepath.Join(dir, "missing")
			fail(ctx, msgpack.EncodedJSON{"path": path}, path+" does not exist.")
		})

		It("Should fail for a file", func(ctx SpecContext) {
			path := filepath.Join(dir, "notes.txt")
			write(path, nil)
			fail(ctx, msgpack.EncodedJSON{"path": path}, path+" is not a folder.")
		})

		It("Should fail for a folder that the Core cannot read", func(ctx SpecContext) {
			if runtime.GOOS == "windows" {
				Skip("Windows gives a folder no read permission")
			}
			if os.Geteuid() == 0 {
				Skip("root reads every folder")
			}
			path := filepath.Join(dir, "locked")
			Expect(os.Mkdir(path, 0)).To(Succeed())
			DeferCleanup(func() { Expect(os.Chmod(path, 0o700)).To(Succeed()) })
			fail(
				ctx,
				msgpack.EncodedJSON{"path": path},
				"The Core cannot read "+path+".",
			)
		})

		It(
			"Should fail for a list that does not finish in time",
			func(ctx SpecContext) {
				write(filepath.Join(dir, "notes.txt"), nil)
				expired, cancel := context.WithDeadline(ctx, time.Now())
				defer cancel()
				stat := b.send(ctx, expired, "list", msgpack.EncodedJSON{"path": dir})
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Message).To(Equal("Failed to list files"))
				Expect(stat.Description).
					To(Equal("The Core did not finish the list of " + dir + " in time."))
			},
		)

		It("Should fail for a path that is not a string", func(ctx SpecContext) {
			fail(ctx, msgpack.EncodedJSON{"path": 12}, "The path must be a string.")
		})
	})
})
