// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  createTestClient,
  DisconnectedError,
  type project,
  type Synnax,
} from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { Project } from "@/platform/project";
import { activeState, savedLayout } from "@/platform/project/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, renderHookWithConsole } from "@/testutil";

const client: Synnax = createTestClient();

describe("Project.useMaybeChange", () => {
  let target: project.Project;
  const layoutKey = id.create();

  beforeAll(async () => {
    target = await client.projects.create({
      name: `proj-${id.create()}`,
      layout: savedLayout(layoutKey),
    });
  });

  it("switches the active project and loads its saved layout", async () => {
    const active = await client.projects.create({
      name: `proj-${id.create()}`,
      layout: Session.Layout.ZERO_SLICE_STATE,
    });
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: { [Session.Project.SLICE_NAME]: activeState(active) },
    });
    const { result } = renderHook(() => Project.useMaybeChange(), { wrapper });

    await act(async () => {
      await result.current(target.key);
    });

    await waitFor(() =>
      expect(Session.Project.selectSelected(store.getState())).toBe(target.key),
    );
    const placed = Session.Layout.select(store.getState(), layoutKey);
    expect(placed?.name).toBe("Operator");
  });

  it("does nothing when the target is already the active project", async () => {
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: { [Session.Project.SLICE_NAME]: activeState(target) },
    });
    const { result } = renderHook(() => Project.useMaybeChange(), { wrapper });
    const before = Session.Layout.selectSliceState(store.getState());

    await act(async () => {
      await result.current(target.key);
    });

    expect(Session.Project.selectSelected(store.getState())).toBe(target.key);
    expect(Session.Layout.selectSliceState(store.getState())).toEqual(before);
  });

  it("throws a DisconnectedError when there is no connected client", async () => {
    const { result } = await renderHookWithConsole(() => Project.useMaybeChange(), {
      client: null,
      preloadedState: {
        [Session.Project.SLICE_NAME]: {
          ...Session.Project.ZERO_SLICE_STATE,
          selected: id.create(),
        },
      },
    });

    let err: unknown;
    await act(async () => {
      try {
        await result.current(target.key);
      } catch (e) {
        err = e;
      }
    });
    expect(DisconnectedError.matches(err)).toBe(true);
  });
});
