// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel, schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Icon, Menu } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import {
  act,
  fireEvent,
  render,
  renderHook,
  screen,
  waitFor,
} from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { describe, expect, it, vi } from "vitest";

import { Panel as PanelFeature } from "@/feature/panel";
import { Panel } from "@/platform/panel";
import {
  createPanelWrapper,
  createServerPanel,
  primePanel,
} from "@/platform/panel/testutil";
import { Session } from "@/session";
import { awaitTextEditingElement } from "@/testutil";

const client = createTestClient();

const resourceTab = (): panel.Tab => ({
  variant: "resource",
  key: uuid.create(),
  resource: schematic.ontologyID(uuid.create()),
});

const viewTab = (): panel.Tab => ({
  variant: "view",
  key: uuid.create(),
  type: "docs",
  args: {},
});

interface Rendered {
  wrapper: FC<PropsWithChildren>;
  panelKey: panel.Key;
}

const setup = async (tabs: panel.Tab[], tabKey: string): Promise<Rendered> => {
  const existing = await createServerPanel(client, { variant: "leaf", tabs });
  const { wrapper } = await createPanelWrapper({
    client,
    panelKey: existing.key,
    tabKey,
  });
  await primePanel(wrapper, existing.key);
  return { wrapper, panelKey: existing.key };
};

const renderMenu = (wrapper: FC<PropsWithChildren>, keys: string[]) =>
  render(
    <Menu.Menu>
      <PanelFeature.TabMenuItems
        keys={keys}
        visible
        position={{ x: 0, y: 0 }}
        cursor={{ x: 0, y: 0 }}
      />
    </Menu.Menu>,
    { wrapper },
  );

describe("Panel.TabMenuItems", () => {
  describe("rename", () => {
    it("offers rename for a resource tab", async () => {
      const tab = resourceTab();
      const { wrapper } = await setup([tab], tab.key);
      renderMenu(wrapper, [tab.key]);
      await waitFor(() => expect(screen.getByText("Rename")).toBeTruthy());
    });

    it("omits rename for a view tab, which has no record to rename", async () => {
      const tab = viewTab();
      const { wrapper } = await setup([tab], tab.key);
      renderMenu(wrapper, [tab.key]);
      await waitFor(() => expect(screen.getByText("Reload console")).toBeTruthy());
      expect(screen.queryByText("Rename")).toBeNull();
    });

    it("starts editing the tab's name", async () => {
      const tab = resourceTab();
      const { wrapper } = await setup([tab], tab.key);
      const service: Panel.EditableTabNameService = {
        useEnsureRetrieved: vi.fn(),
        useSelectName: () => "Resolved Name",
        useRename: () => ({ update: vi.fn() }),
      };
      const Name = Panel.createEditableTabName(service, <Icon.Schematic />);
      render(
        <>
          <Name />
          <Menu.Menu>
            <PanelFeature.TabMenuItems
              keys={[tab.key]}
              visible
              position={{ x: 0, y: 0 }}
              cursor={{ x: 0, y: 0 }}
            />
          </Menu.Menu>
        </>,
        { wrapper },
      );
      await waitFor(() => expect(screen.getByText("Resolved Name")).toBeTruthy());

      fireEvent.click(screen.getByText("Rename"));

      const editing = await awaitTextEditingElement();
      expect(editing.textContent).toEqual("Resolved Name");
    });
  });

  describe("focus", () => {
    it("overlays the tab the menu was opened on, not the panel's front tab", async () => {
      const front = resourceTab();
      const background = resourceTab();
      const { wrapper, panelKey } = await setup([front, background], background.key);

      const { result } = renderHook(
        () => ({
          front: Session.Panel.useSelectIsTabOverlaid(panelKey, front.key),
          background: Session.Panel.useSelectIsTabOverlaid(panelKey, background.key),
        }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.front).toBe(false));

      renderMenu(wrapper, [background.key]);
      await waitFor(() => expect(screen.getByText("Focus")).toBeTruthy());
      await act(async () => {
        fireEvent.click(screen.getByText("Focus"));
      });

      await waitFor(() => expect(result.current.background).toBe(true));
      expect(result.current.front).toBe(false);
    });
  });

  describe("no tab", () => {
    it("offers only a console reload when the menu resolved no tab", async () => {
      const tab = resourceTab();
      const { wrapper } = await setup([tab], tab.key);
      renderMenu(wrapper, []);
      await waitFor(() => expect(screen.getByText("Reload console")).toBeTruthy());
      expect(screen.queryByText("Rename")).toBeNull();
      expect(screen.queryByText("Focus")).toBeNull();
    });
  });
});
