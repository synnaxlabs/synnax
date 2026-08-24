// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { act } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Project } from "@/platform/project";
import { Session } from "@/session";
import { renderHookWithConsole } from "@/testutil";

describe("Project.useMaybeChange", () => {
  it("switches the active project to the given key", async () => {
    const active = id.create();
    const target = id.create();
    const { result, store } = await renderHookWithConsole(
      () => Project.useMaybeChange(),
      {
        preloadedState: {
          [Session.Project.SLICE_NAME]: {
            ...Session.Project.ZERO_SLICE_STATE,
            selected: active,
          },
        },
      },
    );

    act(() => {
      result.current(target);
    });

    expect(Session.Project.selectSelected(store.getState())).toBe(target);
  });

  it("does nothing when the target is already the active project", async () => {
    const active = id.create();
    const { result, store } = await renderHookWithConsole(
      () => Project.useMaybeChange(),
      {
        preloadedState: {
          [Session.Project.SLICE_NAME]: {
            ...Session.Project.ZERO_SLICE_STATE,
            selected: active,
          },
        },
      },
    );
    const before = store.getState();

    act(() => {
      result.current(active);
    });

    expect(store.getState()).toBe(before);
  });
});
