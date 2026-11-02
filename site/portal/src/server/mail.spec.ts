// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { type Notice, render } from "@/server/mail";

const ORIGIN = "https://portal.synnaxlabs.com";

const NOTICE: Notice = {
  subject: "Your Synnax license was revoked",
  heading: "Your license was revoked",
  paragraphs: ["It can no longer activate a Core.", "Reply if this is a mistake."],
  facts: [
    { label: "License", value: "Test rig" },
    { label: "Organization", value: "Acme" },
  ],
  action: { label: "View license", href: "/licenses/abc" },
  reason: "You get this email because you are an admin of Acme.",
};

describe("mail", () => {
  describe("render", () => {
    it("should lay the notice out as plain text", () => {
      expect(render(NOTICE, ORIGIN).text).toBe(
        [
          "Your license was revoked",
          "",
          "It can no longer activate a Core.",
          "",
          "Reply if this is a mistake.",
          "",
          "License: Test rig",
          "Organization: Acme",
          "",
          "View license: https://portal.synnaxlabs.com/licenses/abc",
          "",
          "You get this email because you are an admin of Acme.",
          "Synnax Labs, https://portal.synnaxlabs.com",
        ].join("\n"),
      );
    });

    it("should link the action and the logo on the origin", () => {
      const { html } = render(NOTICE, ORIGIN);
      expect(html.startsWith("<!DOCTYPE html><html")).toBe(true);
      expect(html).toContain('href="https://portal.synnaxlabs.com/licenses/abc"');
      expect(html).toContain('src="https://portal.synnaxlabs.com/email/logo.png"');
    });

    it("should escape names in the HTML", () => {
      const { html } = render(
        { ...NOTICE, facts: [{ label: "Organization", value: "<b>Acme</b> & Co" }] },
        ORIGIN,
      );
      expect(html).toContain("&lt;b&gt;Acme&lt;/b&gt; &amp; Co");
      expect(html).not.toContain("<b>Acme</b>");
    });
  });
});
