// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel, type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Text } from "@synnaxlabs/lyra/text";
import { uuid } from "@synnaxlabs/x";
import { act, render, screen } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Panel } from "@/feature/panel";
import { Panel as PlatformPanel } from "@/platform/panel";
import { createServerPanel } from "@/platform/panel/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, type TestStore } from "@/testutil";

const client = createTestClient();

const ProbeName: PlatformPanel.TabName = () => <Text.Text>probe name</Text.Text>;

const ProbeIcon: PlatformPanel.TabIcon = () => <Text.Text>probe icon</Text.Text>;

const ProbeContent: PlatformPanel.Content = () => <Text.Text>probe content</Text.Text>;

const ProbeToolbar = (): ReactElement => <Text.Text>probe toolbar</Text.Text>;

const REGISTRY: PlatformPanel.Tabs = {
  bare: { Content: ProbeContent, Name: ProbeName, Icon: ProbeIcon },
  configurable: {
    Content: ProbeContent,
    Name: ProbeName,
    Icon: ProbeIcon,
    Toolbar: ProbeToolbar,
  },
};

const setup = async (
  wrapperClient: Synnax | null = client,
): Promise<{ wrapper: FC<PropsWithChildren>; store: TestStore }> => {
  const { wrapper: Console, store } = await createConsoleWrapper({
    client: wrapperClient,
  });
  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <PlatformPanel.RendererContext value={REGISTRY}>
        {children}
      </PlatformPanel.RendererContext>
    </Console>
  );
  return { wrapper, store };
};

const focus = async (store: TestStore, type: string): Promise<panel.Panel> => {
  const tabKey = uuid.create();
  const p = await createServerPanel(client, {
    variant: "leaf",
    tabs: [{ variant: "view", key: tabKey, type }],
  });
  act(() => {
    store.dispatch(Session.Panel.select({ key: p.key }));
    store.dispatch(
      Session.Panel.internalSelectTab({ key: p.key, tabKey, otherTabKeys: [] }),
    );
  });
  return p;
};

describe("Panel.TOOLBAR", () => {
  it("should show an empty state when no tab is focused", async () => {
    const { wrapper } = await setup(null);
    render(Panel.TOOLBAR.content, { wrapper });
    expect(await screen.findByText("No component selected")).toBeTruthy();
  });

  it("should show the tab's name when the tab has no toolbar", async () => {
    const { wrapper, store } = await setup();
    render(Panel.TOOLBAR.content, { wrapper });
    await focus(store, "bare");
    expect(await screen.findByText("probe name")).toBeTruthy();
    expect(screen.getByText("No properties to configure")).toBeTruthy();
  });

  it("should render the tab's own toolbar", async () => {
    const { wrapper, store } = await setup();
    render(Panel.TOOLBAR.content, { wrapper });
    await focus(store, "configurable");
    expect(await screen.findByText("probe toolbar")).toBeTruthy();
  });

  it("should render the focused tab's icon", async () => {
    const { wrapper, store } = await setup();
    render(Panel.TOOLBAR.icon, { wrapper });
    await focus(store, "bare");
    expect(await screen.findByText("probe icon")).toBeTruthy();
  });
});
