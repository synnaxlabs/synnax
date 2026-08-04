// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { Docs } from "@/session/docs";
import { Project } from "@/session/project";
import { Synchronizer } from "@/session/synchronizer";
import { createTestStore } from "@/testutil";

describe("Synchronizer.watch", () => {
  it("should fire on changes to the selected value with next and previous", async () => {
    const store = await createTestStore();
    const onChange = vi.fn();
    Synchronizer.watch(store, (state) => state.project.selected, onChange);
    store.dispatch(Project.select("p1"));
    expect(onChange).toHaveBeenCalledExactlyOnceWith("p1", undefined);
    store.dispatch(Project.select("p2"));
    expect(onChange).toHaveBeenLastCalledWith("p2", "p1");
  });

  it("should not fire when unrelated state changes", async () => {
    const store = await createTestStore();
    const onChange = vi.fn();
    Synchronizer.watch(store, (state) => state.project.selected, onChange);
    store.dispatch(Docs.setLocation({ path: "guide", heading: "intro" }));
    expect(onChange).not.toHaveBeenCalled();
  });

  it("should stop firing after the destructor runs", async () => {
    const store = await createTestStore();
    const onChange = vi.fn();
    const unwatch = Synchronizer.watch(
      store,
      (state) => state.project.selected,
      onChange,
    );
    unwatch();
    store.dispatch(Project.select("p1"));
    expect(onChange).not.toHaveBeenCalled();
  });
});
