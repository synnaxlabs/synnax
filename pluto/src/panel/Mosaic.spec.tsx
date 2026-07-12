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
import { uuid } from "@synnaxlabs/x";
import {
  act,
  fireEvent,
  render,
  type RenderResult,
  waitFor,
} from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { beforeEach, describe, expect, it, type Mock, vi } from "vitest";

import { Errors } from "@/errors";
import { Panel } from "@/panel";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

const ROUND_TRIP = { timeout: 5000 };

const resourceID = (): ontology.ID => ({ type: "lineplot", key: uuid.create() });

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

// contentText is what the harness's children render prop displays for a tab, so
// specs can find a tab's content the way an operator would: by its visible text.
const contentText = ({
  key,
  resource,
  view,
}: { key: string } & Panel.TabContent): string => {
  if (resource != null) return `resource:${resource.key}`;
  if (view != null) return `view:${view.type}`;
  return `empty:${key}`;
};

interface Harness {
  utils: RenderResult;
  children: Mock<(props: Panel.MosaicTabRenderProps) => ReactElement>;
}

const lastContent = (
  children: Harness["children"],
  tabKey: string,
): Panel.MosaicTabRenderProps | undefined =>
  children.mock.calls.filter(([p]) => p.tabKey === tabKey).at(-1)?.[0];

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
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  const renderMosaic = async (
    props: Omit<Panel.MosaicProps, "children">,
  ): Promise<Harness> => {
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

    const children = vi.fn((p: Panel.MosaicTabRenderProps) => (
      <div>{contentText({ key: p.tabKey, resource: p.resource, view: p.view })}</div>
    ));
    let utils!: RenderResult;
    await act(async () => {
      utils = render(
        <Errors.SuspenseBoundary loading={<div>loading</div>}>
          <Panel.Mosaic {...props}>{children}</Panel.Mosaic>
        </Errors.SuspenseBoundary>,
        { wrapper },
      );
    });
    return { utils, children };
  };

  describe("rendering", () => {
    it("should render a tab's content through the children render prop", async () => {
      const tab: panel.TabResource = {
        variant: "resource",
        key: uuid.create(),
        resource: resourceID(),
      };
      const p = await createPanel(tab);
      const { utils, children } = await renderMosaic({ panelKey: p.key });
      await waitFor(() => expect(utils.getByText(contentText(tab))).toBeTruthy());
      expect(lastContent(children, tab.key)?.resource).toEqual(tab.resource);
    });

    it("should resolve each tab's content union", async () => {
      const resourceTab: panel.TabResource = {
        variant: "resource",
        key: uuid.create(),
        resource: resourceID(),
      };
      const viewTab: panel.TabView = {
        variant: "view",
        key: uuid.create(),
        type: "docs",
        name: "",
        args: {},
      };
      const emptyTab: panel.TabEmpty = { variant: "empty", key: uuid.create() };
      const p = await createPanel(resourceTab, viewTab, emptyTab);
      const { utils, children } = await renderMosaic({ panelKey: p.key });
      // Only the selected tab's content is attached to the document; the
      // others render into detached portals and are asserted through the
      // render-prop contract.
      await waitFor(() =>
        expect(utils.getByText(contentText(resourceTab))).toBeTruthy(),
      );

      expect(lastContent(children, resourceTab.key)?.resource).toEqual(
        resourceTab.resource,
      );
      expect(lastContent(children, viewTab.key)?.view?.type).toEqual("docs");
      const empty = lastContent(children, emptyTab.key);
      expect(empty?.resource == null).toBe(true);
      expect(empty?.view == null).toBe(true);
    });

    it("should render tab names through the tabName render prop", async () => {
      const tab: panel.TabResource = {
        variant: "resource",
        key: uuid.create(),
        resource: resourceID(),
      };
      const p = await createPanel(tab);
      const tabName = vi.fn(({ resource }: Panel.MosaicTabNameProps) => (
        <span>{`name:${resource?.key}`}</span>
      ));
      const { utils } = await renderMosaic({ panelKey: p.key, tabName });
      await waitFor(() =>
        expect(utils.getByText(`name:${tab.resource?.key}`)).toBeTruthy(),
      );
    });
  });

  describe("selection", () => {
    it("should select a leaf's first tab when no selection is provided", async () => {
      const a: panel.TabResource = {
        variant: "resource",
        key: uuid.create(),
        resource: resourceID(),
      };
      const b: panel.TabResource = {
        variant: "resource",
        key: uuid.create(),
        resource: resourceID(),
      };
      const p = await createPanel(a, b);
      const { utils, children } = await renderMosaic({ panelKey: p.key });
      await waitFor(() => expect(utils.getByText(contentText(a))).toBeTruthy());
      expect(utils.queryByText(contentText(b))).toBeNull();
      expect(lastContent(children, a.key)?.visible).toBe(true);
      expect(lastContent(children, b.key)?.visible).toBe(false);
    });

    it("should select the most recent tab in selected", async () => {
      const a: panel.TabResource = {
        variant: "resource",
        key: uuid.create(),
        resource: resourceID(),
      };
      const b: panel.TabResource = {
        variant: "resource",
        key: uuid.create(),
        resource: resourceID(),
      };
      const p = await createPanel(a, b);
      const { utils, children } = await renderMosaic({
        panelKey: p.key,
        selected: [b.key, a.key],
      });
      await waitFor(() => expect(utils.getByText(contentText(b))).toBeTruthy());
      expect(lastContent(children, a.key)?.visible).toBe(false);
      expect(lastContent(children, b.key)?.visible).toBe(true);
    });

    it("should select the most recent of each leaf's own tabs", async () => {
      const a1: panel.TabResource = {
        variant: "resource",
        key: uuid.create(),
        resource: resourceID(),
      };
      const a2: panel.TabResource = {
        variant: "resource",
        key: uuid.create(),
        resource: resourceID(),
      };
      const b1: panel.TabResource = {
        variant: "resource",
        key: uuid.create(),
        resource: resourceID(),
      };
      const b2: panel.TabResource = {
        variant: "resource",
        key: uuid.create(),
        resource: resourceID(),
      };
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

      const { utils, children } = await renderMosaic({
        panelKey: p.key,
        selected: [a2.key, b2.key],
      });
      // Both leaves show their own most recent tab, so both are attached to
      // the document.
      await waitFor(() => expect(utils.getByText(contentText(b2))).toBeTruthy());
      expect(utils.getByText(contentText(a2))).toBeTruthy();
      expect(lastContent(children, a1.key)?.visible).toBe(false);
      expect(lastContent(children, b1.key)?.visible).toBe(false);
    });
  });

  describe("gestures", () => {
    it("should remove a tab from the document when its close button is clicked", async () => {
      const a: panel.TabResource = {
        variant: "resource",
        key: uuid.create(),
        resource: resourceID(),
      };
      const b: panel.TabResource = {
        variant: "resource",
        key: uuid.create(),
        resource: resourceID(),
      };
      const p = await createPanel(a, b);
      const { utils } = await renderMosaic({ panelKey: p.key });
      await waitFor(() => expect(utils.getByText(contentText(a))).toBeTruthy());

      const closeButtons = utils.getAllByRole("button", { name: /close/i });
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
      const a: panel.TabResource = {
        variant: "resource",
        key: uuid.create(),
        resource: resourceID(),
      };
      const p = await createPanel(a);
      const onSelect = vi.fn();
      const { utils, children } = await renderMosaic({ panelKey: p.key, onSelect });
      await waitFor(() => expect(utils.getByText(contentText(a))).toBeTruthy());

      await act(async () => {
        fireEvent.click(utils.getByLabelText("pluto-icon--add"));
      });

      await waitFor(() => expect(onSelect).toHaveBeenCalledTimes(1));
      const newKey = onSelect.mock.calls[0][0] as string;
      // The harness doesn't route onSelect into a selected prop, so the new
      // tab renders into a detached portal; assert through the render-prop
      // contract.
      await waitFor(() => expect(lastContent(children, newKey)).toBeDefined());
      const content = lastContent(children, newKey);
      expect(content?.resource == null).toBe(true);
      expect(content?.view == null).toBe(true);
      await waitFor(async () => {
        const fetched = await client.panels.retrieve(p.key);
        expect(panel.findTab(fetched.root, newKey)).not.toBeNull();
      }, ROUND_TRIP);
    });
  });

  describe("cross-client sync", () => {
    it("should render a split dispatched by another client", async () => {
      const a: panel.TabResource = {
        variant: "resource",
        key: uuid.create(),
        resource: resourceID(),
      };
      const b: panel.TabResource = {
        variant: "resource",
        key: uuid.create(),
        resource: resourceID(),
      };
      const p = await createPanel(a, b);
      const { utils } = await renderMosaic({ panelKey: p.key });
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
      const tab: panel.TabEmpty = { variant: "empty", key: uuid.create() };
      const p = await createPanel(tab);
      const { utils } = await renderMosaic({ panelKey: p.key });
      await waitFor(() => expect(utils.getByText(contentText(tab))).toBeTruthy());

      const resource = resourceID();
      await client.panels.dispatch(p.key, "", [
        panel.setTabResource({ key: tab.key, resource }),
      ]);

      await waitFor(
        () =>
          expect(utils.getByText(contentText({ key: tab.key, resource }))).toBeTruthy(),
        ROUND_TRIP,
      );
    });

    it("should drop a tab removed by another client", async () => {
      const a: panel.TabResource = {
        variant: "resource",
        key: uuid.create(),
        resource: resourceID(),
      };
      const b: panel.TabResource = {
        variant: "resource",
        key: uuid.create(),
        resource: resourceID(),
      };
      const p = await createPanel(a, b);
      const { utils } = await renderMosaic({ panelKey: p.key });
      await waitFor(() => expect(utils.getByText(contentText(a))).toBeTruthy());
      expect(utils.getAllByRole("button", { name: /close/i })).toHaveLength(2);

      await client.panels.dispatch(p.key, "", [panel.removeTab({ key: b.key })]);

      await waitFor(
        () => expect(utils.getAllByRole("button", { name: /close/i })).toHaveLength(1),
        ROUND_TRIP,
      );
      expect(utils.getByText(contentText(a))).toBeTruthy();
    });
  });
});
