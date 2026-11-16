// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package console_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing/fstest"

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/console"
	. "github.com/synnaxlabs/x/testutil"
)

type platform struct {
	Signature string `json:"signature"`
	URL       string `json:"url"`
}

type manifest struct {
	Version   string              `json:"version"`
	Notes     string              `json:"notes"`
	PubDate   string              `json:"pub_date"`
	Platforms map[string]platform `json:"platforms"`
}

const (
	releaseURL = "https://github.com/synnaxlabs/synnax/releases/download/" +
		"console/v0.60.1/"
	appFile        = "Synnax.app.tar.gz"
	nsisFile       = "Synnax_0.60.1_x64-setup.exe"
	msiFile        = "Synnax_0.60.1_x64_en-US.msi"
	intelFile      = "Synnax_0.60.1_x64.app.tar.gz"
	linuxSignature = "not_supported_and_tauri_requires_an_entry"
)

// release has the shape of the latest.json of a Console release, plus an Intel macOS
// entry whose file the Core does not embed.
var release = manifest{
	Version: "v0.60.1",
	Notes:   "Synnax Update",
	PubDate: "2026-10-28T22:40:43Z",
	Platforms: map[string]platform{
		"darwin-aarch64": {Signature: "app-signature", URL: releaseURL + appFile},
		"darwin-x86_64":  {Signature: "intel-signature", URL: releaseURL + intelFile},
		"linux-x86_64":   {Signature: linuxSignature, URL: releaseURL + appFile},
		"windows-x86_64": {Signature: "nsis-signature", URL: releaseURL + nsisFile},
		"windows-x86_64-nsis": {
			Signature: "nsis-signature",
			URL:       releaseURL + nsisFile,
		},
		"windows-x86_64-msi": {Signature: "msi-signature", URL: releaseURL + msiFile},
	},
}

var _ = Describe("Update", func() {
	var updateFS fstest.MapFS
	BeforeEach(func() {
		updateFS = fstest.MapFS{
			"latest.json":    {Data: MustSucceed(json.Marshal(release))},
			appFile:          {Data: []byte("app bundle")},
			nsisFile:         {Data: []byte("nsis installer")},
			msiFile:          {Data: []byte("msi installer")},
			appFile + ".sig": {Data: []byte("app-signature")},
			".gitkeep":       {},
		}
	})

	// open also binds the web assets, so every spec checks that the update routes win
	// over the app shell.
	open := func(fsys fstest.MapFS, insecure bool) *fiber.App {
		GinkgoHelper()
		cons := MustSucceed(console.New(console.Config{
			FS:       testFS,
			UpdateFS: fsys,
			Insecure: new(insecure),
		}))
		app := fiber.New()
		cons.BindTo(app)
		return app
	}

	get := func(app *fiber.App, host, path string) *http.Response {
		GinkgoHelper()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Host = host
		return serve(app, req)
	}

	getManifest := func(app *fiber.App, host, forwardedProto string) manifest {
		GinkgoHelper()
		req := httptest.NewRequest(http.MethodGet, "/console/update/latest.json", nil)
		req.Host = host
		if forwardedProto != "" {
			req.Header.Set(fiber.HeaderXForwardedProto, forwardedProto)
		}
		res := serve(app, req)
		Expect(res.StatusCode).To(Equal(http.StatusOK))
		var served manifest
		Expect(json.NewDecoder(res.Body).Decode(&served)).To(Succeed())
		return served
	}

	Describe("Manifest", func() {
		DescribeTable("Should point each platform at the Core that the request names",
			func(insecure bool, host, forwardedProto, base string) {
				served := getManifest(open(updateFS, insecure), host, forwardedProto)
				urls := make(map[string]string, len(served.Platforms))
				for target, p := range served.Platforms {
					urls[target] = p.URL
				}
				Expect(urls).To(Equal(map[string]string{
					"darwin-aarch64":      base + appFile,
					"linux-x86_64":        base + appFile,
					"windows-x86_64":      base + nsisFile,
					"windows-x86_64-nsis": base + nsisFile,
					"windows-x86_64-msi":  base + msiFile,
				}))
			},
			Entry(
				"over HTTP",
				true,
				"192.168.1.20:9090",
				"",
				"http://192.168.1.20:9090/console/update/",
			),
			Entry(
				"over HTTPS",
				false,
				"synnax.example.com",
				"",
				"https://synnax.example.com/console/update/",
			),
			Entry(
				"behind a proxy that ends TLS",
				true,
				"synnax.example.com",
				"https",
				"https://synnax.example.com/console/update/",
			),
			Entry(
				"with an unknown forwarded scheme",
				true,
				"synnax.example.com",
				"ftp",
				"http://synnax.example.com/console/update/",
			),
		)

		It("Should keep the version, notes, publish date, and signatures", func() {
			served := getManifest(open(updateFS, false), "synnax.example.com", "")
			Expect(served.Version).To(Equal(release.Version))
			Expect(served.Notes).To(Equal(release.Notes))
			Expect(served.PubDate).To(Equal(release.PubDate))
			signatures := make(map[string]string, len(served.Platforms))
			for target, p := range served.Platforms {
				signatures[target] = p.Signature
			}
			Expect(signatures).To(Equal(map[string]string{
				"darwin-aarch64":      "app-signature",
				"linux-x86_64":        linuxSignature,
				"windows-x86_64":      "nsis-signature",
				"windows-x86_64-nsis": "nsis-signature",
				"windows-x86_64-msi":  "msi-signature",
			}))
		})

		It("Should drop a platform whose file the Core does not embed", func() {
			served := getManifest(open(updateFS, false), "synnax.example.com", "")
			Expect(served.Platforms).ToNot(HaveKey("darwin-x86_64"))
			Expect(served.Platforms).To(HaveLen(len(release.Platforms) - 1))
		})

		It("Should keep the manifest out of the browser cache", func() {
			res := get(
				open(updateFS, false),
				"synnax.example.com",
				"/console/update/latest.json",
			)
			Expect(res.StatusCode).To(Equal(http.StatusOK))
			Expect(res.Header.Get(fiber.HeaderContentType)).
				To(ContainSubstring(fiber.MIMEApplicationJSON))
			Expect(res.Header.Get(fiber.HeaderCacheControl)).To(Equal("no-store"))
		})
	})

	Describe("Files", func() {
		DescribeTable("Should serve each file that the manifest names",
			func(name string) {
				res := get(
					open(updateFS, false),
					"synnax.example.com",
					"/console/update/"+name,
				)
				Expect(res.StatusCode).To(Equal(http.StatusOK))
				Expect(res.Header.Get(fiber.HeaderCacheControl)).To(Equal("no-store"))
				body := MustSucceed(io.ReadAll(res.Body))
				Expect(body).To(Equal(updateFS[name].Data))
			},
			Entry("the macOS bundle", appFile),
			Entry("the NSIS installer", nsisFile),
			Entry("the MSI installer", msiFile),
		)

		DescribeTable("Should answer 404 for a file that the manifest does not name",
			func(name string) {
				res := get(
					open(updateFS, false),
					"synnax.example.com",
					"/console/update/"+name,
				)
				Expect(res.StatusCode).To(Equal(http.StatusNotFound))
			},
			Entry("an embedded signature", appFile+".sig"),
			Entry("the directory placeholder", ".gitkeep"),
			Entry("the file of a dropped platform", intelFile),
		)
	})

	DescribeTable("Should answer 404 when the Core embeds no update",
		func(names ...string) {
			fsys := fstest.MapFS{}
			for _, name := range names {
				fsys[name] = updateFS[name]
			}
			app := open(fsys, false)
			manifestRes := get(app, "synnax.example.com", "/console/update/latest.json")
			Expect(manifestRes.StatusCode).To(Equal(http.StatusNotFound))
			fileRes := get(app, "synnax.example.com", "/console/update/"+appFile)
			Expect(fileRes.StatusCode).To(Equal(http.StatusNotFound))
		},
		Entry("with no files"),
		Entry("with only the directory placeholder", ".gitkeep"),
		Entry("with a manifest but none of its files", "latest.json", ".gitkeep"),
	)

	DescribeTable("Should fail to open a manifest that cannot be parsed",
		func(data, msg string) {
			Expect(console.New(console.Config{
				UpdateFS: fstest.MapFS{"latest.json": {Data: []byte(data)}},
				Insecure: new(false),
			})).Error().To(MatchError(ContainSubstring(msg)))
		},
		Entry(
			"with malformed JSON",
			"{",
			"failed to parse the Console update manifest",
		),
		Entry(
			"with a malformed URL",
			`{"platforms": {"darwin-aarch64": {"url": "%zz"}}}`,
			"failed to parse the URL of darwin-aarch64",
		),
	)
})
