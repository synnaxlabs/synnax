// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, describe, expect, it } from "vitest";

import { QUERY } from "@/components/platform/choice";

describe("platform choice", () => {
  afterEach(() => delete document.documentElement.dataset.os);

  it.each(["macOS", "Windows", "Linux"])(
    "should show %s first when the page marks it",
    (os) => {
      document.documentElement.dataset.os = os;
      expect(QUERY.initial?.()).toBe(os);
    },
  );

  it("should leave the first tab for an unmarked page", () => {
    expect(QUERY.initial?.()).toBeNull();
  });

  it("should mark the platform the reader picks", () => {
    document.documentElement.dataset.os = "macOS";
    QUERY.onChange?.("Windows");
    expect(document.documentElement.dataset.os).toBe("Windows");
  });

  it("should keep the reader's own platform when they pick Docker", () => {
    document.documentElement.dataset.os = "Linux";
    QUERY.onChange?.("Docker");
    expect(document.documentElement.dataset.os).toBe("Linux");
  });
});
