// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { LETTERS } from "@/letters";
import images from "@/letters/images.json";
import { type Letter, render, UNSUBSCRIBE } from "@/server/letter";

const ORIGIN = "https://portal.synnaxlabs.com";

const LETTER: Letter = {
  version: "0.61.0",
  subject: "Synnax v0.61",
  title: "Arc modules",
  hero: "hero",
  opening: ["Synnax v0.61 is out."],
  features: [
    {
      area: "Arc",
      title: "Arc modules",
      text: "Share code.",
      href: "/reference/control/arc/modules",
      image: "arc",
    },
  ],
  breaking: "Arc syntax changed.",
  closing: ["Emiliano"],
};

const IMAGES = {
  hero: "https://cdn.example.com/hero.jpg",
  arc: "https://cdn.example.com/arc.jpg",
};

describe("letter", () => {
  describe("render", () => {
    it("should lay the letter out as plain text", () => {
      expect(render(LETTER, ORIGIN, IMAGES).text).toBe(
        [
          "Arc modules",
          "",
          "Synnax v0.61 is out.",
          "",
          "Arc: Arc modules",
          "Share code.",
          "https://docs.synnaxlabs.com/reference/control/arc/modules",
          "",
          "Breaking changes. Arc syntax changed.",
          "",
          "Read the release notes: https://docs.synnaxlabs.com/releases#0-61-0",
          "",
          "Emiliano",
          "",
          "You get this email because you have a Synnax account or asked for our " +
            "release notes.",
          `Unsubscribe: ${UNSUBSCRIBE}`,
          `Synnax Labs, ${ORIGIN}`,
        ].join("\n"),
      );
    });

    it("should show the hero, each feature's mail copy, and an unsubscribe link", () => {
      const { html } = render(LETTER, ORIGIN, IMAGES);
      expect(html).toContain('src="https://cdn.example.com/hero.jpg"');
      expect(html).toContain('src="https://cdn.example.com/arc.jpg"');
      expect(html).toContain(`href="${UNSUBSCRIBE}"`);
    });

    it("should leave out the breaking changes when there are none", () => {
      const { text } = render({ ...LETTER, breaking: null }, ORIGIN, IMAGES);
      expect(text).not.toContain("Breaking changes");
    });

    it("should throw when the hero has no mail copy", () => {
      expect(() => render(LETTER, ORIGIN, { arc: IMAGES.arc })).toThrow(
        "letter: hero has no mail copy. Run pnpm mail hero",
      );
    });

    it("should throw when a feature has no mail copy", () => {
      expect(() => render(LETTER, ORIGIN, { hero: IMAGES.hero })).toThrow(
        "letter: arc has no mail copy. Run pnpm mail arc",
      );
    });

    it("should find a mail copy for every image of every letter", () => {
      for (const letter of LETTERS)
        expect(() => render(letter, ORIGIN, images)).not.toThrow();
    });
  });
});
