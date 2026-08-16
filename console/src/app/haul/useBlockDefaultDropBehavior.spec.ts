// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Haul } from "@/app/haul";

const dispatch = (type: "dragover" | "drop"): boolean => {
  const event = new Event(type, { cancelable: true, bubbles: true });
  document.documentElement.dispatchEvent(event);
  return event.defaultPrevented;
};

describe("useBlockDefaultDropBehavior", () => {
  it("should prevent the default dragover behavior while mounted", () => {
    renderHook(() => Haul.useBlockDefaultDropBehavior());
    expect(dispatch("dragover")).toBe(true);
  });

  it("should prevent the default drop behavior while mounted", () => {
    renderHook(() => Haul.useBlockDefaultDropBehavior());
    expect(dispatch("drop")).toBe(true);
  });

  it("should stop preventing default behavior after unmount", () => {
    const { unmount } = renderHook(() => Haul.useBlockDefaultDropBehavior());
    unmount();
    expect(dispatch("dragover")).toBe(false);
    expect(dispatch("drop")).toBe(false);
  });
});
