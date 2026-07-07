// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type ontology, panel } from "@synnaxlabs/client";
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

const resourceTab = (): panel.Tab => ({
  variant: "resource",
  key: uuid.create(),
  resource: { type: "lineplot", key: uuid.create() },
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
      tabs.map((tab) => panel.insertTab({ tab, targetLeaf: panel.ROOT_PATH })),
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
  visible: boolean;
}

// records captures what each tab's content probe resolved from the tab scope, so specs
// can assert content for tabs rendered into detached portals (not in the document).
const records = new Map<string, Recorded>();

// TabContentProbe reads its tab's content from the surrounding tab scope (the content
// delivery contract), records it, and renders the visible content text.
const TabContentProbe = ({
  tabKey,
  visible,
}: Panel.MosaicTabRenderProps): ReactElement => {
  const type = Panel.useSelectTabType({});
  const resource = Panel.useSelectTabResource({});
  const args = Panel.useSelectTabArgs({});
  records.set(tabKey, { type, resource, args, visible });
  const text =
    resource != null
      ? `${resource.type}:${resource.key}`
      : type === "selector"
        ? `empty:${tabKey}`
        : `${type}:${tabKey}`;
  return <div>{text}</div>;
};

// TabNameProbe reads its tab's type from the surrounding tab scope and renders a name,
// exercising the scope-based name delivery.
const TabNameProbe = (): ReactElement => {
  const type = Panel.useSelectTabType({});
  return <span>{`name:${type}`}</span>;
};

const children: Panel.MosaicProps["children"] = ({ tabKey, visible }) => (
  <TabContentProbe tabKey={tabKey} visible={visible} />
);

// Bootstrap pre-warms the flux cache with the suspending hook alone, so the
// mosaic mounts against a cached document. Suspending inside Mosaic itself
// trips a React 19 dev-mode replay bug for hooks declared after the
// suspension point.
const Bootstrap = ({ panelKey }: { panelKey: panel.Key }): ReactElement => {
  Panel.useEnsureRetrieved({ key: panelKey });
  return <p>loaded</p>;
};

describe("Panel.Mosaic", () => {
  let wrapper: FC<PropsWithChildren>;

  beforeEach(async () => {
    records.clear();
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  const renderMosaic = async (
    props: Omit<Panel.MosaicProps, "children"> & { panelKey: panel.Key },
  ): Promise<RenderResult> => {
    let bootstrap!: RenderResult;
    await act(async () => {
      bootstrap = render(
        <Errors.SuspenseBoundary loading={<p>loading</p>}>
          <Bootstrap panelKey={props.panelKey} />
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
          <Panel.Suspended panelKey={props.panelKey}>
            <Panel.Mosaic {...props}>{children}</Panel.Mosaic>
          </Panel.Suspended>
        </Errors.SuspenseBoundary>,
        { wrapper },
      );
    });
    return utils;
  };

  describe("rendering", () => {
    it("should render a tab's content read from the tab scope", async () => {
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

  describe("selection", () => {
    it("should select a leaf's first tab when no selection is provided", async () => {
      const a = resourceTab();
      const b = resourceTab();
      const p = await createPanel(a, b);
      const utils = await renderMosaic({ panelKey: p.key });
      await waitFor(() => expect(utils.getByText(contentText(a))).toBeTruthy());
      expect(utils.queryByText(contentText(b))).toBeNull();
      expect(records.get(a.key)?.visible).toBe(true);
      expect(records.get(b.key)?.visible).toBe(false);
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
      expect(records.get(a.key)?.visible).toBe(false);
      expect(records.get(b.key)?.visible).toBe(true);
    });

    it("should select the most recent of each leaf's own tabs", async () => {
      const a1 = resourceTab();
      const a2 = resourceTab();
      const b1 = resourceTab();
      const b2 = resourceTab();
      const p = await createPanel(a1, a2, b1, b2);
      await client.panels.dispatch(p.key, "", [
        panel.moveTab({ key: b1.key, targetLeaf: panel.ROOT_PATH, location: "right" }),
      ]);
      await client.panels.dispatch(p.key, "", [
        panel.moveTab({
          key: b2.key,
          targetLeaf: panel.childPath(panel.ROOT_PATH, "last"),
        }),
      ]);

      const utils = await renderMosaic({
        panelKey: p.key,
        selected: [a2.key, b2.key],
      });
      // Both leaves show their own most recent tab, so both are attached to
      // the document.
      await waitFor(() => expect(utils.getByText(contentText(b2))).toBeTruthy());
      expect(utils.getByText(contentText(a2))).toBeTruthy();
      expect(records.get(a1.key)?.visible).toBe(false);
      expect(records.get(b1.key)?.visible).toBe(false);
    });
  });

  describe("gestures", () => {
    it("should remove a tab from the document when its close button is clicked", async () => {
      const a = resourceTab();
      const b = resourceTab();
      const p = await createPanel(a, b);
      const utils = await renderMosaic({ panelKey: p.key });
      await waitFor(() => expect(utils.getByText(contentText(a))).toBeTruthy());

      const closeButtons = utils.getAllByLabelText("pluto-tabs__close");
      expect(closeButtons).toHaveLength(2);
      await act(async () => {
        fireEvent.click(closeButtons[0]);
      });

      // Closing the selected tab promotes its sibling to the leaf's selection.
      await waitFor(() => expect(utils.queryByText(contentText(a))).toBeNull());
      await waitFor(() => expect(utils.getByText(contentText(b))).toBeTruthy());
      await waitFor(async () => {
        const fetched = await client.panels.retrieve(p.key);
        expect(panel.findTab(fetched.root, a.key)).toBeNull();
        expect(panel.findTab(fetched.root, b.key)).not.toBeNull();
      }, ROUND_TRIP);
    });

    it("should insert and select a fresh contentless tab from the add button", async () => {
      const a = resourceTab();
      const p = await createPanel(a);
      const onSelect = vi.fn();
      const utils = await renderMosaic({ panelKey: p.key, onSelect });
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
        expect(panel.findTab(fetched.root, newKey)).not.toBeNull();
      }, ROUND_TRIP);
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
        panel.moveTab({ key: b.key, targetLeaf: panel.ROOT_PATH, location: "right" }),
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
      expect(utils.getAllByLabelText("pluto-tabs__close")).toHaveLength(2);

      await client.panels.dispatch(p.key, "", [panel.removeTab({ key: b.key })]);

      await waitFor(
        () => expect(utils.getAllByLabelText("pluto-tabs__close")).toHaveLength(1),
        ROUND_TRIP,
      );
      expect(utils.getByText(contentText(a))).toBeTruthy();
    });
  });
});
