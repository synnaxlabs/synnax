// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { CDN } from "@/components/media/cdn";
import { media } from "@/util/checks/media";

// No page source exists for this route, so failures carry the bare route.
const ROUTE = "/spec/media";
const IMAGE = `${CDN.ROOT}/shot-light.png`;
const VIDEO = `${CDN.ROOT}/clip/abc-light.mp4#t=0.001`;

const check = (html: string): string[] =>
  media(false).page?.({ route: ROUTE, html }) ?? [];

describe("media", () => {
  describe("names", () => {
    it("should pass an image with alt text and a video with a label", () => {
      expect(
        check(
          `<img src="${IMAGE}" alt="A shot">` +
            `<video aria-label="A clip"><source src="${VIDEO}"></video>`,
        ),
      ).toEqual([]);
    });

    it("should pass an image with empty alt text", () => {
      expect(check(`<img src="${IMAGE}" alt="">`)).toEqual([]);
    });

    it("should pass an image with a bare alt attribute", () => {
      expect(check(`<img src="${IMAGE}" alt>`)).toEqual([]);
    });

    it("should fail an image with no alt text", () => {
      expect(check(`<img src="${IMAGE}">`)).toEqual([
        `${ROUTE} - image has no alt text: ${IMAGE}`,
      ]);
    });

    it("should fail a video with no label", () => {
      expect(check(`<video loop><source src="${VIDEO}"></video>`)).toEqual([
        `${ROUTE} - video has no label: ${VIDEO}`,
      ]);
    });
  });
});
