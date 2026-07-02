// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Export } from "@/platform/export";
import { renderHookWithConsole } from "@/testutil/testutil";

// jsdom does not implement the object-URL / anchor-download primitives the browser save
// path relies on. Provide them so the save path completes, and capture the downloaded
// anchor to observe the exported file.
let clickedAnchors: HTMLAnchorElement[] = [];

beforeEach(() => {
  clickedAnchors = [];
  vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:mock");
  vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
  vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(function (
    this: HTMLAnchorElement,
  ) {
    clickedAnchors.push(this);
  });
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("Export.use", () => {
  it("extracts with the key and current store/client, then saves the result", async () => {
    const extract = vi
      .fn()
      .mockResolvedValue({ name: "My Plot", data: '{"key":"plot-1"}' });
    const { result, store } = await renderHookWithConsole(() =>
      Export.use(extract, "lineplot"),
    );
    act(() => result.current("plot-1"));
    await waitFor(() => expect(extract).toHaveBeenCalledTimes(1));
    expect(extract).toHaveBeenCalledWith("plot-1", { store, client: null });
    await waitFor(() => expect(clickedAnchors).toHaveLength(1));
    expect(clickedAnchors[0].download).toEqual("My Plot.json");
  });

  it("does not save anything when the extractor fails", async () => {
    const extract = vi.fn().mockRejectedValue(new Error("cannot extract"));
    const { result } = await renderHookWithConsole(() =>
      Export.use(extract, "lineplot"),
    );
    act(() => result.current("plot-1"));
    await waitFor(() => expect(extract).toHaveBeenCalledTimes(1));
    expect(clickedAnchors).toHaveLength(0);
  });
});
