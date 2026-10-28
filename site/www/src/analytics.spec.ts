// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeAll, beforeEach, describe, expect, it, type Mock, vi } from "vitest";

import { captureClicks } from "@/analytics";

const click = (selector: string): void =>
  document.querySelector<HTMLElement>(selector)!.click();

describe("analytics", () => {
  let posthog: Mock;

  beforeEach(() => {
    posthog = vi.fn();
    vi.stubGlobal("posthog", { capture: posthog });
  });

  describe("captureClicks", () => {
    beforeAll(captureClicks);

    beforeEach(() => {
      document.body.innerHTML = `
        <a class="tracked" data-event="desktop_download_clicked" data-location="hero">
          <span class="label">Download</span>
        </a>
        <a class="plain">Docs</a>
      `;
    });

    it("should capture a click on a tracked element with its location", () => {
      click(".tracked");
      expect(posthog).toHaveBeenCalledExactlyOnceWith(
        "desktop_download_clicked",
        { location: "hero" },
        { transport: "sendBeacon" },
      );
    });

    it("should capture a click inside a tracked element", () => {
      click(".label");
      expect(posthog).toHaveBeenCalledOnce();
    });

    it("should capture nothing for an untracked element", () => {
      click(".plain");
      expect(posthog).not.toHaveBeenCalled();
    });
  });
});
