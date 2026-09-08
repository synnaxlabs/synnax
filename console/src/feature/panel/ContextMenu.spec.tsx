// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  group,
  panel,
  project,
  schematic,
  type Synnax as Client,
} from "@synnaxlabs/client";
import {
  type BuiltInRole,
  createTestClient,
  RoleClients,
} from "@synnaxlabs/client/testutil";
import { Drift } from "@synnaxlabs/drift";
import { Icon, Menu } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import {
  act,
  fireEvent,
  render,
  renderHook,
  type RenderResult,
  screen,
  waitFor,
} from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "web" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { Panel as PanelFeature } from "@/feature/panel";
import { Modals } from "@/platform/modals";
import { Panel } from "@/platform/panel";
import {
  createPanelWrapper,
  createServerPanel,
  primePanel,
} from "@/platform/panel/testutil";
import { Session } from "@/session";
import {
  awaitTextEditingElement,
  createTestClientWithGrants,
  type TestStore,
  uniqueName,
} from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

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
  store: TestStore;
}

const setup = async (tabs: panel.Tab[], tabKey: string): Promise<Rendered> => {
  const existing = await createServerPanel(client, { variant: "leaf", tabs });
  const { wrapper, store } = await createPanelWrapper({
    client,
    panelKey: existing.key,
    tabKey,
  });
  await primePanel(wrapper, existing.key);
  return { wrapper, panelKey: existing.key, store };
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
  beforeEach(() => {
    mocks.engine = "web";
  });

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
      await waitFor(() => expect(screen.getByText("Reload Console")).toBeTruthy());
      expect(screen.queryByText("Rename")).toBeNull();
    });

    it("starts editing the tab's name", async () => {
      const tab = resourceTab();
      const { wrapper } = await setup([tab], tab.key);
      const service: Panel.EditableTabNameService = {
        useEnsure: vi.fn(),
        useName: () => "Resolved Name",
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

  describe("focus hints", () => {
    const hintOf = (label: string): string => {
      const item = screen.getByText(label).closest("button");
      const hint = item?.querySelector('[aria-label="trigger-indicator"]');
      return hint?.textContent?.toLowerCase() ?? "";
    };

    it("names the key that performs the action the entry offers", async () => {
      const front = resourceTab();
      const { wrapper } = await setup([front], front.key);
      renderMenu(wrapper, [front.key]);
      await waitFor(() => expect(screen.getByText("Focus")).toBeTruthy());
      expect(hintOf("Focus")).toContain("l");
      await act(async () => {
        fireEvent.click(screen.getByText("Focus"));
      });
      await waitFor(() => expect(screen.getByText("Exit focus")).toBeTruthy());
      expect(hintOf("Exit focus")).toContain("l");
    });

    it("withholds every hint from a menu open on a tab that is not focused", async () => {
      const front = resourceTab();
      const background = resourceTab();
      const { wrapper, panelKey, store } = await setup(
        [front, background],
        background.key,
      );
      act(() => {
        store.dispatch(
          Session.Panel.internalSelectTab({
            key: panelKey,
            tabKey: front.key,
            otherTabKeys: [],
          }),
        );
      });
      renderMenu(wrapper, [background.key]);
      await waitFor(() => expect(screen.getByText("Focus")).toBeTruthy());
      expect(hintOf("Focus")).toEqual("");
      expect(hintOf("Close")).toEqual("");
      expect(hintOf("Rename")).toEqual("");
    });
  });

  describe("no tab", () => {
    it("offers only a console reload when the menu resolved no tab", async () => {
      const tab = resourceTab();
      const { wrapper } = await setup([tab], tab.key);
      renderMenu(wrapper, []);
      await waitFor(() => expect(screen.getByText("Reload Console")).toBeTruthy());
      expect(screen.queryByText("Rename")).toBeNull();
      expect(screen.queryByText("Focus")).toBeNull();
    });
  });

  describe("move to new window", () => {
    it("hides the item in the browser, which cannot open a window", async () => {
      const tab = resourceTab();
      const { wrapper } = await setup([tab], tab.key);
      renderMenu(wrapper, [tab.key]);
      await waitFor(() => expect(screen.getByText("Move to panel")).toBeTruthy());
      expect(screen.queryByText("Move to new window")).toBeNull();
    });

    it("mints a panel holding the tab and opens a window showing it", async () => {
      mocks.engine = "tauri";
      const moved = resourceTab();
      const stays = resourceTab();
      const existing = await createServerPanel(client, {
        variant: "leaf",
        tabs: [moved, stays],
      });
      const { wrapper, store } = await createPanelWrapper({
        client,
        panelKey: existing.key,
        tabKey: moved.key,
      });
      await primePanel(wrapper, existing.key);

      const menu = renderMenu(wrapper, [moved.key]);
      await waitFor(() => expect(screen.getByText("Move to new window")).toBeTruthy());
      await act(async () => {
        fireEvent.click(screen.getByText("Move to new window"));
      });
      // The production menu closes on click; the harness unmounts it so the menu's
      // tab-scoped selectors don't re-evaluate against the now-removed tab.
      menu.unmount();

      let windowKey!: string;
      await waitFor(() => {
        const state = store.getState();
        const win = Drift.selectWindows(state).find(
          ({ key, reserved }) => key !== Drift.MAIN_WINDOW && reserved,
        );
        expect(win).toBeDefined();
        windowKey = win!.key;
        expect(state.panels.windows[windowKey]?.selected).toBeDefined();
      });

      const minted = store.getState().panels.windows[windowKey].selected!;
      await waitFor(async () => {
        const [src, dst] = await Promise.all([
          client.panels.retrieve(existing.key),
          client.panels.retrieve(minted),
        ]);
        expect(panel.findTab(src.root, moved.key)).toBeUndefined();
        expect(panel.findTab(src.root, stays.key)).toBeDefined();
        expect(panel.findTab(dst.root, moved.key)).toEqual(moved);
      });
    });
  });

  describe("move to panel", () => {
    it("opens the picker on the tab the menu was opened on", async () => {
      const projectKey = (
        await client.projects.create({ name: uniqueName("project"), layout: {} })
      ).key;
      const parent = project.ontologyID(projectKey);
      const [front, moved] = [resourceTab(), resourceTab()];
      const source = await client.panels.create({
        key: uuid.create(),
        name: uniqueName("panel"),
        parent,
        // The menu opens on the second tab, so a picker fed the panel's front tab
        // instead moves the wrong one.
        root: { variant: "leaf", tabs: [front, moved] },
      });
      const destination = await client.panels.create({
        key: uuid.create(),
        name: uniqueName("panel"),
        parent,
        root: { variant: "leaf", tabs: [] },
      });
      const { wrapper } = await createPanelWrapper({
        client,
        project: projectKey,
        panelKey: source.key,
        tabKey: moved.key,
      });

      let rerender!: RenderResult["rerender"];
      await act(async () => {
        ({ rerender } = render(
          <>
            <Menu.Menu>
              <PanelFeature.TabMenuItems
                keys={[moved.key]}
                visible
                position={{ x: 0, y: 0 }}
                cursor={{ x: 0, y: 0 }}
              />
            </Menu.Menu>
            <Modals.Stack />
          </>,
          { wrapper },
        ));
      });
      await screen.findByText("Move to panel");
      await act(async () => {
        fireEvent.click(screen.getByText("Move to panel"));
      });

      const row = await screen.findByText(destination.name);
      // The production menu closes on click; the harness drops it so the menu's
      // tab-scoped selectors don't re-evaluate against the moved tab. The empty slot
      // holds the stack's position, which remounting would tear the open modal down.
      rerender(
        <>
          {null}
          <Modals.Stack />
        </>,
      );
      await act(async () => {
        fireEvent.click(row);
      });
      await waitFor(async () => {
        const [src, dst] = await Promise.all([
          client.panels.retrieve(source.key),
          client.panels.retrieve(destination.key),
        ]);
        expect(panel.findTab(dst.root, moved.key)).toEqual(moved);
        expect(panel.findTab(src.root, moved.key)).toBeUndefined();
        expect(panel.findTab(src.root, front.key)).toBeDefined();
      });
    });
  });
});

describe("Panel.TabMenuItems permissions", () => {
  const setupAs = async (
    as: Client,
    tab: panel.Tab,
  ): Promise<FC<PropsWithChildren>> => {
    const existing = await createServerPanel(client, { variant: "leaf", tabs: [tab] });
    const { key } = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const { wrapper } = await createPanelWrapper({
      client: as,
      project: key,
      panelKey: existing.key,
      tabKey: tab.key,
    });
    await primePanel(wrapper, existing.key);
    return wrapper;
  };

  beforeEach(() => {
    mocks.engine = "tauri";
  });

  it.each<BuiltInRole>(["Viewer", "Operator"])(
    "should withhold rename and both moves from a %s",
    async (role) => {
      const tab = resourceTab();
      const wrapper = await setupAs(await roles.get(role), tab);
      renderMenu(wrapper, [tab.key]);
      await waitFor(() => expect(screen.getByText("Reload Console")).toBeTruthy());
      expect(screen.queryByText("Rename")).toBeNull();
      expect(screen.queryByText("Move to panel")).toBeNull();
      expect(screen.queryByText("Move to new window")).toBeNull();
    },
  );

  it("should offer rename and both moves to an engineer", async () => {
    const tab = resourceTab();
    const wrapper = await setupAs(await roles.get("Engineer"), tab);
    renderMenu(wrapper, [tab.key]);
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(await screen.findByText("Move to panel")).toBeTruthy();
    expect(await screen.findByText("Move to new window")).toBeTruthy();
  });

  it("should withhold the tear-off from a subject who cannot create panels", async () => {
    const tab = resourceTab();
    const editor = await createTestClientWithGrants(client, {
      retrieve: [
        panel.TYPE_ONTOLOGY_ID,
        project.TYPE_ONTOLOGY_ID,
        schematic.TYPE_ONTOLOGY_ID,
        group.TYPE_ONTOLOGY_ID,
      ],
      update: [panel.TYPE_ONTOLOGY_ID, schematic.TYPE_ONTOLOGY_ID],
    });
    const wrapper = await setupAs(editor, tab);
    renderMenu(wrapper, [tab.key]);
    expect(await screen.findByText("Move to panel")).toBeTruthy();
    expect(screen.queryByText("Move to new window")).toBeNull();
  });
});
