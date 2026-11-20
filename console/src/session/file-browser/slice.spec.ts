// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { FileBrowser } from "@/session/file-browser";

const add = (state: FileBrowser.SliceState, folder: string, rack = 1) =>
  FileBrowser.reducer(state, FileBrowser.addRecent({ rack, folder }));

describe("FileBrowser slice", () => {
  it("should move a folder picked again to the front of its rack", () => {
    let state = add(FileBrowser.ZERO_SLICE_STATE, "/a");
    state = add(state, "/b");
    state = add(state, "/a");
    state = add(state, "/c", 2);
    expect(state.recent).toEqual({ 1: ["/a", "/b"], 2: ["/c"] });
  });

  it("should keep the five newest folders of a rack", () => {
    let state = FileBrowser.ZERO_SLICE_STATE;
    for (const f of ["/1", "/2", "/3", "/4", "/5", "/6"]) state = add(state, f);
    expect(state.recent[1]).toEqual(["/6", "/5", "/4", "/3", "/2"]);
  });

  it("should remove a folder from its rack only", () => {
    let state = add(FileBrowser.ZERO_SLICE_STATE, "/a");
    state = add(state, "/b");
    state = add(state, "/a", 2);
    state = FileBrowser.reducer(
      state,
      FileBrowser.removeRecent({ rack: 1, folder: "/a" }),
    );
    expect(state.recent).toEqual({ 1: ["/b"], 2: ["/a"] });
  });
});
