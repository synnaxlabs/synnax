// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Drift } from "@synnaxlabs/drift";
import { Icon, Panel as PPanel, Text } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { act, render, screen, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement, useEffect } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Mosaic } from "@/feature/panel/Mosaic";
import { Panel } from "@/platform/panel";
import { createServerPanel } from "@/platform/panel/testutil";
import { Session } from "@/session";
import {
  type ConsolePreloadedState,
  createConsoleWrapper,
  type TestStore,
} from "@/testutil";

// The shape a reload hydrates: the persisted selection with the keep-alive set
// cleared, since it is excluded from persistence.
const hydrated = (key: panel.Key): ConsolePreloadedState => ({
  [Session.Panel.SLICE_NAME]: {
    windows: {
      [Drift.MAIN_WINDOW]: { ...Session.Panel.ZERO_WINDOW_STATE, selected: key },
    },
  },
});

const client = createTestClient();

const mounts: panel.Key[] = [];
const unmounts: panel.Key[] = [];

const ProbeContent: Panel.Content = () => {
  const panelKey = PPanel.useKey();
  useEffect(() => {
    mounts.push(panelKey);
    return () => void unmounts.push(panelKey);
  }, [panelKey]);
  return <Text.Text>{`content-${panelKey}`}</Text.Text>;
};

const ProbeName: Panel.TabName = () => <Text.Text>probe</Text.Text>;

const ProbeIcon: Panel.TabIcon = () => <Icon.Visualize />;

const REGISTRY: Panel.Tabs = {
  probe: { Content: ProbeContent, Name: ProbeName, Icon: ProbeIcon },
};

const createTab = (): panel.NewTab => ({
  variant: "view",
  key: uuid.create(),
  type: "probe",
});

const probePanel = async (): Promise<panel.Panel> =>
  await createServerPanel(client, {
    variant: "leaf",
    tabs: [{ variant: "view", key: uuid.create(), type: "probe" }],
  });

const setup = async (
  preloadedState?: ConsolePreloadedState,
): Promise<{
  wrapper: FC<PropsWithChildren>;
  store: TestStore;
}> => {
  const { wrapper: Console, store } = await createConsoleWrapper({
    client,
    preloadedState,
  });
  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Panel.RendererContext value={REGISTRY}>{children}</Panel.RendererContext>
    </Console>
  );
  return { wrapper, store };
};

const mountCount = (key: panel.Key): number => mounts.filter((k) => k === key).length;

describe("Panel.Mosaic keep-alive", () => {
  beforeEach(() => {
    mounts.length = 0;
    unmounts.length = 0;
  });

  it("should keep a panel mounted and reattach it across switches", async () => {
    const a = await probePanel();
    const b = await probePanel();
    const { wrapper, store } = await setup();
    render(<Mosaic onCreateTab={createTab} />, { wrapper });

    act(() => {
      store.dispatch(Session.Panel.select({ key: a.key }));
    });
    await waitFor(() => expect(screen.getByText(`content-${a.key}`)).toBeTruthy());
    expect(mountCount(a.key)).toBe(1);

    act(() => {
      store.dispatch(Session.Panel.select({ key: b.key }));
    });
    await waitFor(() => expect(screen.getByText(`content-${b.key}`)).toBeTruthy());
    // The first panel leaves the document (its portal element detaches) but its
    // React tree stays mounted.
    expect(screen.queryByText(`content-${a.key}`)).toBeNull();
    expect(unmounts).toHaveLength(0);

    act(() => {
      store.dispatch(Session.Panel.select({ key: a.key }));
    });
    await waitFor(() => expect(screen.getByText(`content-${a.key}`)).toBeTruthy());
    // Reattached, not remounted.
    expect(mountCount(a.key)).toBe(1);
    expect(unmounts).toHaveLength(0);
  });

  // Regression: the keep-alive set is excluded from persistence, so a reload
  // restored a selected panel with nothing portaled in and the mosaic came up
  // blank until the user picked another panel.
  it("should render a panel selected before a reload", async () => {
    const a = await probePanel();
    const { wrapper } = await setup(hydrated(a.key));
    render(<Mosaic onCreateTab={createTab} />, { wrapper });

    await waitFor(() => expect(screen.getByText(`content-${a.key}`)).toBeTruthy());
  });

  it("should show the empty state while keeping visited panels mounted", async () => {
    const a = await probePanel();
    const { wrapper, store } = await setup();
    render(<Mosaic onCreateTab={createTab} />, { wrapper });

    act(() => {
      store.dispatch(Session.Panel.select({ key: a.key }));
    });
    await waitFor(() => expect(screen.getByText(`content-${a.key}`)).toBeTruthy());

    act(() => {
      store.dispatch(Session.Panel.clearSelected({}));
    });
    await waitFor(() =>
      expect(
        screen.getByText("No panels open. Create one to get started."),
      ).toBeTruthy(),
    );
    expect(unmounts).toHaveLength(0);
  });

  // Without this the keep-alive set grows for the window's whole lifetime, and
  // every visited panel keeps streaming its channels with nothing on screen.
  it("should unmount the least recently selected panel past the cap", async () => {
    const panels: panel.Panel[] = [];
    for (let i = 0; i < 6; i++) panels.push(await probePanel());
    const { wrapper, store } = await setup();
    render(<Mosaic onCreateTab={createTab} />, { wrapper });

    for (const { key } of panels) {
      act(() => {
        store.dispatch(Session.Panel.select({ key }));
      });
      await waitFor(() => expect(screen.getByText(`content-${key}`)).toBeTruthy());
    }

    // The first five stay mounted through the sixth selection; only the least
    // recently selected one is released.
    expect(unmounts).toEqual([panels[0].key]);
    expect(mountCount(panels[1].key)).toBe(1);
  });

  it("should unmount a panel deleted while another is selected", async () => {
    const a = await probePanel();
    const b = await probePanel();
    const { wrapper, store } = await setup();
    render(<Mosaic onCreateTab={createTab} />, { wrapper });

    act(() => {
      store.dispatch(Session.Panel.select({ key: a.key }));
    });
    await waitFor(() => expect(screen.getByText(`content-${a.key}`)).toBeTruthy());
    act(() => {
      store.dispatch(Session.Panel.select({ key: b.key }));
    });
    await waitFor(() => expect(screen.getByText(`content-${b.key}`)).toBeTruthy());

    act(() => {
      store.dispatch(Session.Panel.remove(a.key));
    });
    await waitFor(() => expect(unmounts).toEqual([a.key]));
    expect(screen.getByText(`content-${b.key}`)).toBeTruthy();
  });
});
