// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { Flux, type Pluto, Schematic as PSchematic, Status } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { act, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { useHandleNodeClickAction } from "@/feature/schematic/navigate";
import {
  client,
  createPreloadedState,
  createSchematic,
} from "@/feature/schematic/testutil";
import { Session } from "@/session";
import { renderHookWithConsole, uniqueName } from "@/testutil";

interface RenderNavigateArgs {
  configs?: Record<string, schematic.ElementConfig>;
  editable?: boolean;
}

const renderNavigateHook = async ({
  configs = {},
  editable = false,
}: RenderNavigateArgs) => {
  const source = await createSchematic({
    nodes: Object.keys(configs).map((key) => ({ key, position: { x: 0, y: 0 } })),
    configs,
  });
  const hook = await renderHookWithConsole(
    () => ({
      handler: useHandleNodeClickAction(source.key),
      fluxStore: Flux.useStore<Pluto.FluxStore>(),
      notifications: Status.useNotifications(),
    }),
    { client, preloadedState: createPreloadedState(source.key, { editable }) },
  );
  await act(async () => {
    await PSchematic.retrieveSingle({
      store: hook.result.current.fluxStore,
      client,
      query: { key: source.key },
    });
  });
  return { source, ...hook };
};

const offPageConfig = (
  page: string,
  overrides: Partial<schematic.ElementConfigOffPageReference> = {},
): schematic.ElementConfig => ({
  variant: "off_page_reference",
  label: { label: "Target Ref" },
  page,
  ...overrides,
});

const expectPlacedLayout = async (
  store: { getState: () => Session.State },
  target: schematic.Schematic,
) => {
  await waitFor(() => {
    const layout = Session.Layout.select(store.getState(), target.key);
    if (layout == null) throw new Error("layout not placed");
    expect(layout.name).toBe(target.name);
  });
};

describe("Schematic.useHandleNodeClickAction", () => {
  it("navigates to the referenced schematic on double click", async () => {
    const target = await createSchematic({ name: uniqueName("target") });
    const { result, store } = await renderNavigateHook({
      configs: { n1: offPageConfig(target.key) },
    });
    act(() => result.current.handler("n1", true));
    await expectPlacedLayout(store, target);
  });

  it("ignores single clicks when double-click navigation is the default", async () => {
    const target = await createSchematic({ name: uniqueName("target") });
    const control = await createSchematic({ name: uniqueName("control") });
    const { result, store } = await renderNavigateHook({
      configs: {
        n1: offPageConfig(target.key),
        ctl: offPageConfig(control.key),
      },
    });
    act(() => result.current.handler("n1", false));
    act(() => result.current.handler("ctl", true));
    await expectPlacedLayout(store, control);
    expect(Session.Layout.select(store.getState(), target.key)).toBeUndefined();
  });

  it("navigates on single click when dblClickNav is disabled", async () => {
    const target = await createSchematic({ name: uniqueName("target") });
    const { result, store } = await renderNavigateHook({
      configs: { n1: offPageConfig(target.key, { dblClickNav: false }) },
    });
    act(() => result.current.handler("n1", false));
    await expectPlacedLayout(store, target);
  });

  it("does not navigate while the schematic is editable", async () => {
    const target = await createSchematic({ name: uniqueName("target") });
    const control = await createSchematic({ name: uniqueName("control") });
    const { source, result, store } = await renderNavigateHook({
      configs: {
        n1: offPageConfig(target.key),
        ctl: offPageConfig(control.key),
      },
      editable: true,
    });
    act(() => result.current.handler("n1", true));
    act(() => {
      store.dispatch(
        Session.Schematic.setEditable({ key: source.key, editable: false }),
      );
    });
    act(() => result.current.handler("ctl", true));
    await expectPlacedLayout(store, control);
    expect(Session.Layout.select(store.getState(), target.key)).toBeUndefined();
  });

  it("ignores nodes that are not off-page references", async () => {
    const target = await createSchematic({ name: uniqueName("target") });
    const control = await createSchematic({ name: uniqueName("control") });
    const { result, store } = await renderNavigateHook({
      configs: {
        n1: { variant: "valve" },
        ctl: offPageConfig(control.key),
      },
    });
    act(() => result.current.handler("n1", true));
    act(() => result.current.handler("ctl", true));
    await expectPlacedLayout(store, control);
    expect(Session.Layout.select(store.getState(), target.key)).toBeUndefined();
  });

  it("ignores references with an empty page", async () => {
    const control = await createSchematic({ name: uniqueName("control") });
    const { result, store } = await renderNavigateHook({
      configs: { n1: offPageConfig(""), ctl: offPageConfig(control.key) },
    });
    act(() => result.current.handler("n1", true));
    act(() => result.current.handler("ctl", true));
    await expectPlacedLayout(store, control);
    expect(
      result.current.notifications.statuses.filter((st) => st.variant === "error"),
    ).toHaveLength(0);
  });

  it("raises an error status naming the reference when the target is missing", async () => {
    const { result } = await renderNavigateHook({
      configs: { n1: offPageConfig(uuid.create()) },
    });
    act(() => result.current.handler("n1", true));
    await waitFor(() =>
      expect(
        result.current.notifications.statuses.some(
          (st) =>
            st.variant === "error" && st.message === 'Schematic "Target Ref" not found',
        ),
      ).toBe(true),
    );
  });
});
