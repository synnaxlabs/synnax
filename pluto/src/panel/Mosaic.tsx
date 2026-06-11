// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, panel } from "@synnaxlabs/client";
import { type location, uuid } from "@synnaxlabs/x";
import { type ComponentType, type ReactElement, useCallback, useMemo } from "react";

import { type Component } from "@/component";
import { context } from "@/context";
import { Mosaic as Base } from "@/mosaic";
import {
  type TabContent,
  useDispatch,
  useEnsureRetrieved,
  useSelectRoot,
  useSelectTabContent,
} from "@/panel/queries";
import { Portal } from "@/portal";
import { Tabs } from "@/tabs";

// MosaicTabRenderProps is the contract for the children (content) render prop. A
// tab's content is a union: a resource (content with a backing document) or an
// inline view. The consumer decides how to render each; the panel layer is
// intentionally renderer-agnostic so different consumers (console, integration
// tests, future apps) plug in their own resolution.
export interface MosaicTabRenderProps extends TabContent {
  tabKey: string;
  visible: boolean;
}

// MosaicTabNameProps is the contract for the tabName render prop. It extends the
// base tab-name props with the tab's content union so the consumer can resolve a
// display name (and rename behavior) from the underlying resource or view.
export interface MosaicTabNameProps extends Tabs.NameProps, TabContent {}

// MosaicProps mirrors Base.MosaicProps where the panel-aware shell adds value
// (panelKey, the resource-aware children render prop) and otherwise passes
// presentation slots straight through to Base.Mosaic.
export interface MosaicProps extends Pick<
  Base.MosaicProps,
  | "rounded"
  | "bordered"
  | "borderColor"
  | "background"
  | "contextMenu"
  | "emptyContent"
  | "addTooltip"
  | "className"
  | "onFileDrop"
> {
  panelKey: panel.Key;
  // focused is the tab the operator is focused on, or absent when focus is
  // elsewhere. It only drives the focus accent; a focused tab is expected to
  // sit at the head of selected. Held externally so the session-level "what is
  // the operator looking at" state lives outside the server-synced panel
  // document.
  focused?: string;
  // selected is the operator's selection memory: tab keys ordered most
  // recently selected first. Leaf selection derives from it alone: each leaf
  // shows the most recent of its own tabs, so selecting a tab in one leaf does
  // not snap sibling leaves back to their first tab.
  selected?: string[];
  // onSelect fires when the operator clicks a tab. The panel-aware shell does
  // not persist this selection; consumers route it into their session-state store.
  onSelect?: (tabKey: string) => void;
  // children renders a tab's content from its resolved content union.
  children: (props: MosaicTabRenderProps) => ReactElement | null;
  // tabName renders a tab's display name from its content union: name text plus
  // any passive indicators (icon, unsaved-changes dot), subscribing to the
  // underlying resource as needed. When omitted, tab strips show the default
  // (empty) name. Rename, when supported, is the consumer's to wire through the
  // underlying content (e.g. renaming the resource), so the panel layer needs no
  // rename prop of its own.
  tabName?: (props: MosaicTabNameProps) => ReactElement | null;
}

// adaptToMosaic walks the typed panel tree and produces the Base.Node shape
// with path-derived numeric keys (root = 1, first child = 2k, last child =
// 2k + 1).
const adaptToMosaic = (root: panel.Node, selected: string[] | undefined): Base.Node => {
  const preference = selected ?? [];
  const visit = (node: panel.Node | undefined, key: number): Base.Node => {
    if (node == null) return { key };
    if (node.split != null)
      return {
        key,
        direction: node.split.direction,
        size: node.split.size,
        first: visit(node.split.first, panel.childPath(key, "first")),
        last: visit(node.split.last, panel.childPath(key, "last")),
      };

    if (node.leaf == null) return { key, tabs: [] };
    const tabs: Tabs.Tab[] = node.leaf.tabs.map((t) => ({
      tabKey: t.key,
      name: "",
      closable: true,
      // Resource and view tabs carry a renameable name; empty (selector) tabs do not.
      editable: t.resource != null || t.view != null,
    }));
    const selected =
      preference.find((key) => tabs.some((t) => t.tabKey === key)) ?? tabs[0]?.tabKey;
    return { key, tabs, selected };
  };
  return visit(root, panel.ROOT_PATH);
};

interface TabNameContextValue {
  panelKey: panel.Key;
  tabName?: Component.RenderProp<MosaicTabNameProps>;
}

const [TabNameContext, useTabNameContext] = context.create<TabNameContextValue>({
  displayName: "Panel.Mosaic.TabContext.Provider",
  providerName: "Panel.Mosaic.TabContext.Provider",
});

const TabName: ComponentType<Tabs.NameProps> = (props) => {
  const { tabKey } = props;
  const { panelKey: key, tabName } = useTabNameContext("Panel.Mosaic.TabName");
  const content = useSelectTabContent({ key, tabKey });
  if (tabName == null) return <Tabs.DefaultName {...props} />;
  return tabName({ ...props, ...content });
};

interface ContentProps extends Pick<MosaicProps, "children"> {
  panelKey: panel.Key;
  tabKey: string;
  visible: boolean;
}

const Content = ({
  panelKey: key,
  tabKey,
  visible,
  children,
}: ContentProps): ReactElement | null =>
  children({ ...useSelectTabContent({ key, tabKey }), tabKey, visible });

export const Mosaic = ({
  panelKey: key,
  focused,
  selected,
  onSelect,
  children,
  tabName,
  ...rest
}: MosaicProps): ReactElement | null => {
  useEnsureRetrieved({ key });
  const { dispatch } = useDispatch();
  const treeRoot = useSelectRoot({ key });
  const root = useMemo(() => adaptToMosaic(treeRoot, selected), [treeRoot, selected]);

  const handleDrop = useCallback(
    (
      targetLeaf: number,
      tabKey: string,
      location: location.Location,
      index?: number,
    ) => {
      const action = panel.moveTab({ key: tabKey, targetLeaf, index, location });
      dispatch({ key, actions: [action] });
    },
    [dispatch, key],
  );

  const handleResize = useCallback(
    (split: number, size: number) =>
      dispatch({ key, actions: [panel.resizeSplit({ split, size })] }),
    [dispatch, key],
  );

  const handleClose = useCallback(
    (tabKey: string) => dispatch({ key, actions: [panel.removeTab({ key: tabKey })] }),
    [dispatch, key],
  );

  const handleCreate = useCallback(
    (node: number, loc: location.Location, tabKeys?: string[]) => {
      let tabs: panel.Tab[];
      if (tabKeys == null) tabs = [{ key: uuid.create() }];
      else
        tabs = tabKeys.flatMap((raw) => {
          const parsed = ontology.idZ.safeParse(raw);
          return parsed.success ? [{ key: uuid.create(), resource: parsed.data }] : [];
        });
      const restLeaf =
        loc === "center" ? node : panel.childPath(node, panel.splitSide(loc));
      const actions = tabs.map((tab, i) =>
        panel.insertTab({
          tab,
          targetLeaf: i === 0 ? node : restLeaf,
          location: i === 0 ? loc : undefined,
        }),
      );
      dispatch({ key, actions });
      onSelect?.(tabs[tabs.length - 1].key);
    },
    [dispatch, key, onSelect],
  );

  const [portalRef, portalNodes] = Base.usePortal({
    root,
    onSelect,
    children: ({ tabKey, visible }) => (
      <Content panelKey={key} tabKey={tabKey} visible={visible !== false}>
        {children}
      </Content>
    ),
  });

  const tabNameContext = useMemo<TabNameContextValue>(
    () => ({ panelKey: key, tabName }),
    [key, tabName],
  );

  const renderProp = useCallback<Tabs.RenderProp>(
    (props) => {
      const node = portalRef.current.get(props.tabKey);
      if (node == null) return null;
      return <Portal.Out node={node} />;
    },
    [portalRef],
  );

  return (
    <TabNameContext value={tabNameContext}>
      {portalNodes}
      <Base.Mosaic
        {...rest}
        root={root}
        activeTab={focused}
        onSelect={onSelect}
        onDrop={handleDrop}
        onResize={handleResize}
        onClose={handleClose}
        onCreate={handleCreate}
        Name={tabName != null ? TabName : undefined}
      >
        {renderProp}
      </Base.Mosaic>
    </TabNameContext>
  );
};
