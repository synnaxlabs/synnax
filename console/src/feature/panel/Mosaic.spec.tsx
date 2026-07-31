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
import { Icon, Panel as PPanel, Text } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { act, render, screen, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement, useEffect } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Mosaic } from "@/feature/panel/Mosaic";
import { Panel } from "@/platform/panel";
import { createServerPanel } from "@/platform/panel/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, type TestStore } from "@/testutil";

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

const setup = async (): Promise<{
  wrapper: FC<PropsWithChildren>;
  store: TestStore;
}> => {
  const { wrapper: Console, store } = await createConsoleWrapper({ client });
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
    await waitFor(() => expect(screen.getByText("No panels open.")).toBeTruthy());
    expect(unmounts).toHaveLength(0);
  });
});
