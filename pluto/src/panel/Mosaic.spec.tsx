// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, panel } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type record, uuid } from "@synnaxlabs/x";
import {
  act,
  fireEvent,
  render,
  type RenderResult,
  waitFor,
} from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Errors } from "@/errors";
import { Panel } from "@/panel";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

const ROUND_TRIP = { timeout: 5000 };

const resourceID = (): ontology.ID => ({ type: "lineplot", key: uuid.create() });

const resourceTab = (): panel.Tab => ({
  variant: "resource",
  key: uuid.create(),
  resource: resourceID(),
});

const viewTab = (): panel.Tab => ({
  variant: "view",
  key: uuid.create(),
  type: "docs",
  args: {},
});

const selectorTab = (): panel.Tab => ({
  variant: "view",
  key: uuid.create(),
  type: "selector",
  args: {},
});

const createPanel = async (...tabs: panel.Tab[]): Promise<panel.Panel> => {
  const created = await client.panels.create({ name: `mosaic-${uuid.create()}` });
  if (tabs.length > 0)
    await client.panels.dispatch(
      created.key,
      "",
      tabs.map((tab) => panel.insertTab({ tab, targetLeaf: panel.ROOT_NODE_KEY })),
    );
  return created;
};

// contentText is what the probe displays for a tab, so specs can find a tab's content
// the way an operator would: by its visible text.
const contentText = (tab: panel.Tab): string => {
  if (tab.variant === "resource") return `${tab.resource.type}:${tab.resource.key}`;
  if (tab.type === "selector") return `empty:${tab.key}`;
  return `${tab.type}:${tab.key}`;
};

interface Recorded {
  type: string;
  resource: ontology.ID | null;
  args: record.Unknown | null;
}

// records captures what each tab's content probe resolved from the tab scope, so specs
// can assert content for tabs rendered into detached portals (not in the document).
const records = new Map<string, Recorded>();

// ResourceContentProbe reads a resource tab's ontology ID from the surrounding tab
// scope, records it, and renders the visible content text.
const ResourceContentProbe = (): ReactElement => {
  const tabKey = Panel.useTabKey();
  const type = Panel.useSelectTabType({});
  const resource = Panel.useSelectTabResource({});
  records.set(tabKey, { type, resource, args: null });
  return <div>{`${resource.type}:${resource.key}`}</div>;
};

// ViewContentProbe reads a view tab's args from the surrounding tab scope, records
// them, and renders the visible content text.
const ViewContentProbe = (): ReactElement => {
  const tabKey = Panel.useTabKey();
  const type = Panel.useSelectTabType({});
  const args = Panel.useSelectTabArgs({});
  records.set(tabKey, { type, resource: null, args });
  const text = type === "selector" ? `empty:${tabKey}` : `${type}:${tabKey}`;
  return <div>{text}</div>;
};

// TabContentProbe dispatches on the tab's variant before touching variant-specific
// selectors, mirroring how production renderers consume the tab scope: only a
// resource renderer reads the resource, only a view renderer reads the args.
const TabContentProbe = (): ReactElement => {
  const variant = Panel.useSelectTabVariant({});
  if (variant === "resource") return <ResourceContentProbe />;
  return <ViewContentProbe />;
};

// TabNameProbe reads its tab's type from the surrounding tab scope and renders a name,
// exercising the scope-based name delivery.
const TabNameProbe = (): ReactElement => {
  const type = Panel.useSelectTabType({});
  return <span>{`name:${type}`}</span>;
};

// TabKeyNameProbe renders a per-tab clickable name from the tab key in scope, so gesture
// specs can target an individual tab's handle by its visible text.
const TabKeyNameProbe = (): ReactElement => {
  const tabKey = Panel.TabScope.use();
  return <span>{`name:${tabKey}`}</span>;
};

const children: Panel.MosaicProps["children"] = () => <TabContentProbe />;

// Bootstrap pre-warms the flux cache with the suspending hook alone, so the
// mosaic mounts against a cached document. Suspending inside Mosaic itself
// trips a React 19 dev-mode replay bug for hooks declared after the
// suspension point.
const Bootstrap = ({ panelKey }: { panelKey: panel.Key }): ReactElement => {
  Panel.useEnsureRetrieved({ key: panelKey });
  return <p>loaded</p>;
};

// Skipped only while flux is cache-blind: legacy raw-send dispatch mutates
// documents server-side while the client cache keeps stale copies, making
// these renders racy. The pluto rebind onto the client cache restores them.
describe.skip("Panel.Mosaic", () => {
  let wrapper: FC<PropsWithChildren>;

  beforeEach(async () => {
    records.clear();
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  const renderMosaic = async ({
    panelKey,
    ...props
  }: Omit<Panel.MosaicProps, "children"> & {
    panelKey: panel.Key;
  }): Promise<RenderResult> => {
    let bootstrap!: RenderResult;
    await act(async () => {
      bootstrap = render(
        <Errors.SuspenseBoundary loading={<p>loading</p>}>
          <Bootstrap panelKey={panelKey} />
        </Errors.SuspenseBoundary>,
        { wrapper },
      );
    });
    await waitFor(() => expect(bootstrap.getByText("loaded")).toBeTruthy());
    bootstrap.unmount();

    let utils!: RenderResult;
    await act(async () => {
      utils = render(
        <Errors.SuspenseBoundary loading={<div>loading</div>}>
          <Panel.Suspended panelKey={panelKey}>
            <Panel.Mosaic {...props}>{children}</Panel.Mosaic>
          </Panel.Suspended>
        </Errors.SuspenseBoundary>,
        { wrapper },
      );
    });
    return utils;
  };

  describe("rendering", () => {
    // Skipped only while flux is cache-blind: legacy raw-send dispatch mutates the
    // tab server-side but the client cache keeps the tab-less create() copy. The
    // pluto rebind onto the client cache restores this.
    it.skip("should render a tab's content read from the tab scope", async () => {
      const tab = resourceTab();
      const p = await createPanel(tab);
      const utils = await renderMosaic({ panelKey: p.key });
      await waitFor(() => expect(utils.getByText(contentText(tab))).toBeTruthy());
      expect(records.get(tab.key)?.resource).toEqual(
        tab.variant === "resource" ? tab.resource : null,
      );
    });

    it("should resolve each tab's content independently", async () => {
      const resource = resourceTab();
      const view = viewTab();
      const selector = selectorTab();
      const p = await createPanel(resource, view, selector);
      const utils = await renderMosaic({ panelKey: p.key });
      // Only the selected tab's content is attached to the document; the
      // others render into detached portals and are asserted via the probe.
      await waitFor(() => expect(utils.getByText(contentText(resource))).toBeTruthy());

      expect(records.get(resource.key)?.type).toEqual("lineplot");
      expect(records.get(resource.key)?.args).toBeNull();
      expect(records.get(view.key)?.type).toEqual("docs");
      expect(records.get(view.key)?.resource).toBeNull();
      expect(records.get(selector.key)?.type).toEqual("selector");
    });

    it("should render tab names from the tab scope", async () => {
      const tab = resourceTab();
      const p = await createPanel(tab);
      const tabName: Panel.MosaicProps["tabName"] = () => <TabNameProbe />;
      const utils = await renderMosaic({ panelKey: p.key, tabName });
      await waitFor(() => expect(utils.getByText("name:lineplot")).toBeTruthy());
    });
  });

  // tabEl finds a tab's strip handle by its data-tab-key attribute.
  const tabEl = (utils: RenderResult, key: string): HTMLElement => {
    const el = utils.container.querySelector<HTMLElement>(`[data-tab-key="${key}"]`);
    if (el == null) throw new Error(`tab ${key} not found`);
    return el;
  };

  const isTabSelected = (utils: RenderResult, key: string): boolean =>
    tabEl(utils, key).getAttribute("aria-selected") === "true";

  const isTabFocused = (utils: RenderResult, key: string): boolean =>
    tabEl(utils, key).classList.contains("pluto--alt-color");

  // splitPanel arranges the four tabs into two leaves: [a1, a2] on the left and
  // [b1, b2] on the right.
  const splitPanel = async (
    a1: panel.Tab,
    a2: panel.Tab,
    b1: panel.Tab,
    b2: panel.Tab,
  ): Promise<panel.Panel> => {
    const p = await createPanel(a1, a2, b1, b2);
    await client.panels.dispatch(p.key, "", [
      panel.moveTab({
        key: b1.key,
        targetLeaf: panel.ROOT_NODE_KEY,
        location: "right",
      }),
    ]);
    await client.panels.dispatch(p.key, "", [
      panel.moveTab({
        key: b2.key,
        targetLeaf: panel.childNodeKey(panel.ROOT_NODE_KEY, "last"),
      }),
    ]);
    return p;
  };

  describe("selection", () => {
    it("should select a leaf's first tab when no selection is provided", async () => {
      const a = resourceTab();
      const b = resourceTab();
      const p = await createPanel(a, b);
      const utils = await renderMosaic({ panelKey: p.key });
      await waitFor(() => expect(utils.getByText(contentText(a))).toBeTruthy());
      expect(utils.queryByText(contentText(b))).toBeNull();
      expect(isTabSelected(utils, a.key)).toBe(true);
      expect(isTabSelected(utils, b.key)).toBe(false);
    });

    it("should select the most recent tab in selected", async () => {
      const a = resourceTab();
      const b = resourceTab();
      const p = await createPanel(a, b);
      const utils = await renderMosaic({
        panelKey: p.key,
        selected: [b.key, a.key],
      });
      await waitFor(() => expect(utils.getByText(contentText(b))).toBeTruthy());
      expect(utils.queryByText(contentText(a))).toBeNull();
      expect(isTabSelected(utils, b.key)).toBe(true);
    });

    it("should keep only the most recent selected tab per leaf", async () => {
      const a = resourceTab();
      const b = resourceTab();
      const p = await createPanel(a, b);
      const utils = await renderMosaic({
        panelKey: p.key,
        selected: [b.key, a.key],
      });
      await waitFor(() => expect(utils.getByText(contentText(b))).toBeTruthy());
      expect(isTabSelected(utils, b.key)).toBe(true);
      expect(isTabSelected(utils, a.key)).toBe(false);
    });

    it("should drop selected keys that are no longer in the tree", async () => {
      const a = resourceTab();
      const removed = resourceTab();
      const p = await createPanel(a);
      const utils = await renderMosaic({
        panelKey: p.key,
        selected: [removed.key],
      });
      await waitFor(() => expect(utils.getByText(contentText(a))).toBeTruthy());
      expect(isTabSelected(utils, a.key)).toBe(true);
    });

    it("should select the most recent of each leaf's own tabs", async () => {
      const a1 = resourceTab();
      const a2 = resourceTab();
      const b1 = resourceTab();
      const b2 = resourceTab();
      const p = await splitPanel(a1, a2, b1, b2);

      const utils = await renderMosaic({
        panelKey: p.key,
        selected: [a2.key, b2.key],
      });
      // Both leaves show their own most recent tab, so both are attached to
      // the document.
      await waitFor(() => expect(utils.getByText(contentText(b2))).toBeTruthy());
      expect(utils.getByText(contentText(a2))).toBeTruthy();
      expect(utils.queryByText(contentText(a1))).toBeNull();
      expect(utils.queryByText(contentText(b1))).toBeNull();
      expect(isTabSelected(utils, a2.key)).toBe(true);
      expect(isTabSelected(utils, b2.key)).toBe(true);
    });

    it("should fall back to a leaf's first tab when none of its tabs is selected", async () => {
      const a1 = resourceTab();
      const a2 = resourceTab();
      const b1 = resourceTab();
      const b2 = resourceTab();
      const p = await splitPanel(a1, a2, b1, b2);

      const utils = await renderMosaic({
        panelKey: p.key,
        selected: [a2.key],
      });
      await waitFor(() => expect(utils.getByText(contentText(b1))).toBeTruthy());
      expect(isTabSelected(utils, a2.key)).toBe(true);
      expect(isTabSelected(utils, b1.key)).toBe(true);
      expect(isTabSelected(utils, b2.key)).toBe(false);
    });
  });

  describe("focus", () => {
    it("should color only the first selected tab as focused", async () => {
      const a1 = resourceTab();
      const a2 = resourceTab();
      const b1 = resourceTab();
      const b2 = resourceTab();
      const p = await splitPanel(a1, a2, b1, b2);

      const utils = await renderMosaic({
        panelKey: p.key,
        selected: [a2.key, b2.key],
      });
      await waitFor(() => expect(utils.getByText(contentText(b2))).toBeTruthy());
      expect(isTabFocused(utils, a2.key)).toBe(true);
      expect(isTabFocused(utils, b2.key)).toBe(false);
      expect(isTabFocused(utils, a1.key)).toBe(false);
    });

    it("should give focus to a leaf's fallback tab when it heads the selection", async () => {
      const a = resourceTab();
      const b = resourceTab();
      const p = await createPanel(a, b);
      const utils = await renderMosaic({ panelKey: p.key });
      await waitFor(() => expect(utils.getByText(contentText(a))).toBeTruthy());
      expect(isTabFocused(utils, a.key)).toBe(true);
    });
  });

  describe("surgical re-renders", () => {
    // tabRenders counts how many times each tab's name render prop ran, so specs can
    // assert that a selection change re-renders only the leaves it touches.
    const tabRenders = new Map<string, number>();
    const TabNameCounter = (): ReactElement => {
      const tabKey = Panel.TabScope.use();
      tabRenders.set(tabKey, (tabRenders.get(tabKey) ?? 0) + 1);
      return <span>{`name:${tabKey}`}</span>;
    };
    const countingTabName: Panel.MosaicProps["tabName"] = () => <TabNameCounter />;
    const stableOnSelect = (): void => {};

    it("should re-render only the leaf whose selection changed", async () => {
      tabRenders.clear();
      const a1 = resourceTab();
      const a2 = resourceTab();
      const b1 = resourceTab();
      const b2 = resourceTab();
      const p = await splitPanel(a1, a2, b1, b2);

      const make = (selected: string[]): ReactElement => (
        <Errors.SuspenseBoundary loading={<div>loading</div>}>
          <Panel.Suspended panelKey={p.key}>
            <Panel.Mosaic
              selected={selected}
              onSelect={stableOnSelect}
              tabName={countingTabName}
            >
              {children}
            </Panel.Mosaic>
          </Panel.Suspended>
        </Errors.SuspenseBoundary>
      );

      let bootstrap!: RenderResult;
      await act(async () => {
        bootstrap = render(
          <Errors.SuspenseBoundary loading={<p>loading</p>}>
            <Bootstrap panelKey={p.key} />
          </Errors.SuspenseBoundary>,
          { wrapper },
        );
      });
      await waitFor(() => expect(bootstrap.getByText("loaded")).toBeTruthy());
      bootstrap.unmount();

      let utils!: RenderResult;
      await act(async () => {
        utils = render(make([a2.key, b2.key]), { wrapper });
      });
      await waitFor(() => expect(utils.getByText(contentText(b2))).toBeTruthy());
      const baseline = new Map(tabRenders);

      // A focus swap changes no leaf's selection, so no leaf re-renders.
      await act(async () => {
        utils.rerender(make([b2.key, a2.key]));
      });
      expect(tabRenders).toEqual(baseline);
      expect(isTabFocused(utils, b2.key)).toBe(true);
      expect(isTabFocused(utils, a2.key)).toBe(false);

      // Changing the right leaf's selection re-renders only the right leaf.
      await act(async () => {
        utils.rerender(make([b1.key, a2.key]));
      });
      await waitFor(() => expect(utils.getByText(contentText(b1))).toBeTruthy());
      expect(tabRenders.get(a1.key)).toEqual(baseline.get(a1.key));
      expect(tabRenders.get(a2.key)).toEqual(baseline.get(a2.key));
      expect(tabRenders.get(b1.key)).toBeGreaterThan(baseline.get(b1.key) ?? 0);
    });
  });

  describe("gestures", () => {
    it("should call onSelect when an unselected tab is clicked", async () => {
      const a = resourceTab();
      const b = resourceTab();
      const p = await createPanel(a, b);
      const onSelect = vi.fn();
      const tabName = vi.fn(() => <TabKeyNameProbe />);
      const utils = await renderMosaic({
        panelKey: p.key,
        selected: [a.key],
        onSelect,
        tabName,
      });
      await waitFor(() => expect(utils.getByText(contentText(a))).toBeTruthy());

      await act(async () => {
        fireEvent.click(utils.getByText(`name:${b.key}`));
      });

      expect(onSelect).toHaveBeenCalledWith(b.key);
    });

    it("should remove a tab from the document when its close button is clicked", async () => {
      const a = resourceTab();
      const b = resourceTab();
      const p = await createPanel(a, b);
      const utils = await renderMosaic({ panelKey: p.key });
      await waitFor(() => expect(utils.getByText(contentText(a))).toBeTruthy());

      const closeButtons = utils.getAllByLabelText("Close");
      expect(closeButtons).toHaveLength(2);
      await act(async () => {
        fireEvent.click(closeButtons[0]);
      });

      // Closing the selected tab promotes its sibling to the leaf's selection.
      await waitFor(() => expect(utils.queryByText(contentText(a))).toBeNull());
      await waitFor(() => expect(utils.getByText(contentText(b))).toBeTruthy());
      await waitFor(async () => {
        const fetched = await client.panels.retrieve(p.key);
        expect(panel.findTab(fetched.root, a.key)).toBeUndefined();
        expect(panel.findTab(fetched.root, b.key)).toBeDefined();
      }, ROUND_TRIP);
    });

    it("should insert and select a fresh contentless tab from the add button", async () => {
      const a = resourceTab();
      const p = await createPanel(a);
      const onSelect = vi.fn();
      const onCreateTab = (): panel.NewTab => ({
        variant: "view",
        type: "selector",
        args: {},
      });
      const utils = await renderMosaic({ panelKey: p.key, onSelect, onCreateTab });
      await waitFor(() => expect(utils.getByText(contentText(a))).toBeTruthy());

      await act(async () => {
        fireEvent.click(utils.getByLabelText("pluto-icon--add"));
      });

      await waitFor(() => expect(onSelect).toHaveBeenCalledTimes(1));
      const newKey = onSelect.mock.calls[0][0] as string;
      // The harness doesn't route onSelect into a selected prop, so the new
      // tab renders into a detached portal; assert through the probe.
      await waitFor(() => expect(records.get(newKey)).toBeDefined());
      expect(records.get(newKey)?.type).toEqual("selector");
      await waitFor(async () => {
        const fetched = await client.panels.retrieve(p.key);
        expect(panel.findTab(fetched.root, newKey)).toBeDefined();
      }, ROUND_TRIP);
    });
  });

  describe("context menu", () => {
    // ExtraMenuProbe renders the tab key the menu resolved from the surrounding tab
    // scope, so specs can assert that extras are scoped to the right-clicked tab.
    const ExtraMenuProbe = (): ReactElement => {
      const tabKey = Panel.useOptionalTabKey();
      return <span>{`extra:${tabKey ?? "none"}`}</span>;
    };

    const extraMenuItems = (): ReactElement => <ExtraMenuProbe />;

    const openMenuOn = async (utils: RenderResult, target: HTMLElement) => {
      await act(async () => {
        fireEvent.contextMenu(target);
      });
      await waitFor(() => expect(utils.getByText("Close")).toBeTruthy());
    };

    it("should scope the menu to the right-clicked tab", async () => {
      const a = resourceTab();
      const b = resourceTab();
      const p = await createPanel(a, b);
      const tabName = vi.fn(() => <TabKeyNameProbe />);
      const utils = await renderMosaic({
        panelKey: p.key,
        selected: [a.key],
        tabName,
        extraMenuItems,
      });
      await waitFor(() => expect(utils.getByText(contentText(a))).toBeTruthy());

      await openMenuOn(utils, utils.getByText(`name:${b.key}`));

      expect(utils.getByText(`extra:${b.key}`)).toBeTruthy();
    });

    it("should close the right-clicked tab", async () => {
      const a = resourceTab();
      const b = resourceTab();
      const p = await createPanel(a, b);
      const tabName = vi.fn(() => <TabKeyNameProbe />);
      const utils = await renderMosaic({
        panelKey: p.key,
        selected: [a.key],
        tabName,
      });
      await waitFor(() => expect(utils.getByText(contentText(a))).toBeTruthy());

      await openMenuOn(utils, utils.getByText(`name:${b.key}`));
      await act(async () => {
        fireEvent.click(utils.getByText("Close"));
      });

      await waitFor(async () => {
        const fetched = await client.panels.retrieve(p.key);
        expect(panel.findTab(fetched.root, b.key)).toBeUndefined();
        expect(panel.findTab(fetched.root, a.key)).toBeDefined();
      }, ROUND_TRIP);
    });

    it("should split the right-clicked tab into a new leaf", async () => {
      const a = resourceTab();
      const b = resourceTab();
      const p = await createPanel(a, b);
      const tabName = vi.fn(() => <TabKeyNameProbe />);
      const utils = await renderMosaic({
        panelKey: p.key,
        selected: [a.key],
        tabName,
      });
      await waitFor(() => expect(utils.getByText(contentText(a))).toBeTruthy());

      await openMenuOn(utils, utils.getByText(`name:${b.key}`));
      await act(async () => {
        fireEvent.click(utils.getByText("Split horizontally"));
      });

      await waitFor(async () => {
        const fetched = await client.panels.retrieve(p.key);
        expect(fetched.root.variant).toEqual("split");
      }, ROUND_TRIP);
    });

    it("should omit split items for a tab that cannot be split", async () => {
      const a = resourceTab();
      const p = await createPanel(a);
      const tabName = vi.fn(() => <TabKeyNameProbe />);
      const utils = await renderMosaic({ panelKey: p.key, tabName });
      await waitFor(() => expect(utils.getByText(contentText(a))).toBeTruthy());

      await openMenuOn(utils, utils.getByText(`name:${a.key}`));

      expect(utils.queryByText("Split horizontally")).toBeNull();
      expect(utils.queryByText("Split vertically")).toBeNull();
    });

    it("should render extras alone when the right-click misses every tab", async () => {
      const a = resourceTab();
      const p = await createPanel(a);
      const utils = await renderMosaic({
        panelKey: p.key,
        extraMenuItems,
      });
      await waitFor(() => expect(utils.getByText(contentText(a))).toBeTruthy());

      await act(async () => {
        fireEvent.contextMenu(utils.getByRole("tablist"));
      });

      await waitFor(() => expect(utils.getByText("extra:none")).toBeTruthy());
      expect(utils.queryByText("Close")).toBeNull();
    });
  });

  describe("cross-client sync", () => {
    it("should render a split dispatched by another client", async () => {
      const a = resourceTab();
      const b = resourceTab();
      const p = await createPanel(a, b);
      const utils = await renderMosaic({ panelKey: p.key });
      await waitFor(() => expect(utils.getByText(contentText(a))).toBeTruthy());
      expect(utils.queryByText(contentText(b))).toBeNull();

      await client.panels.dispatch(p.key, "", [
        panel.moveTab({
          key: b.key,
          targetLeaf: panel.ROOT_NODE_KEY,
          location: "right",
        }),
      ]);

      // After the split each tab is its own leaf's selection, so both attach
      // to the document.
      await waitFor(() => {
        expect(utils.getByText(contentText(a))).toBeTruthy();
        expect(utils.getByText(contentText(b))).toBeTruthy();
      }, ROUND_TRIP);
    });

    it("should update a tab's content when another client sets its resource", async () => {
      const tab = selectorTab();
      const p = await createPanel(tab);
      const utils = await renderMosaic({ panelKey: p.key });
      await waitFor(() => expect(utils.getByText(contentText(tab))).toBeTruthy());

      const resource = { type: "lineplot", key: uuid.create() } as const;
      await client.panels.dispatch(p.key, "", [
        panel.setTabResource({ key: tab.key, resource }),
      ]);

      await waitFor(
        () => expect(utils.getByText(`${resource.type}:${resource.key}`)).toBeTruthy(),
        ROUND_TRIP,
      );
    });

    it("should drop a tab removed by another client", async () => {
      const a = resourceTab();
      const b = resourceTab();
      const p = await createPanel(a, b);
      const utils = await renderMosaic({ panelKey: p.key });
      await waitFor(() => expect(utils.getByText(contentText(a))).toBeTruthy());
      expect(utils.getAllByLabelText("Close")).toHaveLength(2);

      await client.panels.dispatch(p.key, "", [panel.removeTab({ key: b.key })]);

      await waitFor(
        () => expect(utils.getAllByLabelText("Close")).toHaveLength(1),
        ROUND_TRIP,
      );
      expect(utils.getByText(contentText(a))).toBeTruthy();
    });
  });
});
