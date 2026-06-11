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

import { context } from "@/context";
import { Flux } from "@/flux";
import { Mosaic as Base } from "@/mosaic";
import {
  type FluxSubStore,
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
export interface MosaicTabRenderProps {
  tabKey: string;
  // resource and view are both null when the tab has no content yet. The consumer
  // renders its selector in that case and swaps in content via SetTabResource /
  // SetTabView. At most one is non-null.
  resource: ontology.ID | null;
  view: panel.TabView | null;
  visible: boolean;
}

// MosaicTabNameProps is the contract for the tabName render prop. It extends the
// base tab-name props with the tab's content union so the consumer can resolve a
// display name (and rename behavior) from the underlying resource or view.
export interface MosaicTabNameProps extends Tabs.NameProps {
  resource: ontology.ID | null;
  view: panel.TabView | null;
}

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
  // The tab the consumer wants displayed as active. Held externally so the
  // session-level "what is the operator looking at" state lives outside the
  // server-synced panel document.
  activeTab?: string;
  // recentTabs is the consumer's most-recently-active tab keys (most recent
  // first). Each leaf selects the most recent of its tabs, so activating a tab
  // in one leaf does not snap sibling leaves back to their first tab.
  recentTabs?: string[];
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
const adaptToMosaic = (
  root: panel.Node | undefined,
  activeTab: string | undefined,
  recentTabs: string[] | undefined,
): Base.Node => {
  // Selection preference per leaf: the active tab first, then the MRU.
  const preference =
    activeTab != null ? [activeTab, ...(recentTabs ?? [])] : (recentTabs ?? []);
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

// TabNameContext carries the panel key and the consumer's tabName resolver down
// to the Base.Mosaic tab strip, which only exposes Tabs.NameProps. A
// module-stable Name component reads it so tab names never remount on tree
// changes.
interface TabNameContextValue {
  panelKey: panel.Key | null;
  tabName?: (props: MosaicTabNameProps) => ReactElement | null;
}

const [TabNameContext, useTabNameContext] = context.create<TabNameContextValue>({
  displayName: "Panel.Mosaic.TabName",
  defaultValue: { panelKey: null },
});

const TabName: ComponentType<Tabs.NameProps> = (props) => {
  const { panelKey, tabName } = useTabNameContext();
  const content = useSelectTabContent({ key: panelKey ?? "", tabKey: props.tabKey });
  if (tabName == null) return <Tabs.DefaultName {...props} />;
  return tabName({
    ...props,
    resource: content?.resource ?? null,
    view: content?.view ?? null,
  });
};

interface ContentProps {
  panelKey: panel.Key;
  tabKey: string;
  visible: boolean;
  children: MosaicProps["children"];
}

// Content subscribes to a single tab's content union, so a resource or view
// change re-renders only this tab.
const Content = ({
  panelKey,
  tabKey,
  visible,
  children,
}: ContentProps): ReactElement | null => {
  const content = useSelectTabContent({ key: panelKey, tabKey });
  if (content == null) return null;
  return children({ tabKey, resource: content.resource, view: content.view, visible });
};

// Mosaic renders a Flux-backed Panel as a Base.Mosaic. The architecture is a
// direct parallel of the legacy console layout mosaic; the only material
// difference is that gestures dispatch panel actions through the SY-3038
// substrate instead of mutating a Redux slice. Other connected consoles
// observe each mutation through the panel action channel and apply it via
// the panel reducer, so cross-client sync is automatic.
//
// Mosaic suspends while the panel document loads; wrap it in a Suspense
// boundary.
//
// Tab content is delivered through the children render prop, which receives the
// tabKey and its resolved content union. The portal pattern borrowed from
// Base.Mosaic preserves content lifetime across structural changes, so moving a
// tab between leaves does not unmount its content.
//
// Tab display names are the consumer's responsibility through the tabName render
// prop, which receives the same content union. The default tab name on the
// underlying Tabs.Tab is empty; consumers resolve a name (and rename) from the
// referenced resource or view.
export const Mosaic = ({
  panelKey,
  activeTab,
  recentTabs,
  onSelect,
  children,
  tabName,
  ...rest
}: MosaicProps): ReactElement | null => {
  useEnsureRetrieved({ key: panelKey });
  const store = Flux.useStore<FluxSubStore>();
  const { dispatch } = useDispatch();
  const treeRoot = useSelectRoot({ key: panelKey });
  const root = useMemo(
    () => adaptToMosaic(treeRoot, activeTab, recentTabs),
    [treeRoot, activeTab, recentTabs],
  );

  const handleSelect = useCallback((tabKey: string) => onSelect?.(tabKey), [onSelect]);

  // handleDrop maps Base.Mosaic's drop gesture onto panel actions. A center drop
  // moves (or reorders) the tab into the target leaf; an edge drop splits the leaf
  // and moves the tab into the new sibling, mirroring handleCreate's composition.
  const handleDrop = useCallback(
    (key: number, tabKey: string, loc: location.Location, index?: number) => {
      const actions: panel.Action[] = [];
      let targetLeaf = key;
      if (loc !== "center") {
        // Splitting a leaf against its own only tab is a degenerate gesture: the
        // result would be the tab next to an empty pane. Treat it as a no-op.
        const treeRoot = store.panels.get(panelKey)?.root;
        const sourceLeaf =
          treeRoot != null ? panel.tabLeafPath(treeRoot, tabKey) : null;
        const sourceTabCount =
          sourceLeaf != null
            ? (panel.walkPath(treeRoot, sourceLeaf)?.leaf?.tabs.length ?? 0)
            : 0;
        if (sourceLeaf === key && sourceTabCount <= 1) return;
        actions.push(panel.splitLeaf({ leaf: key, location: loc, size: 0.5 }));
        const side = loc === "left" || loc === "top" ? "first" : "last";
        targetLeaf = panel.childPath(key, side);
      }
      actions.push(panel.moveTab({ key: tabKey, targetLeaf, index: index ?? 0 }));
      dispatch({ key: panelKey, actions });
    },
    [dispatch, panelKey, store],
  );

  // Resize.useMultiple emits on mount as well as on drags, and a stale emission
  // can land after the panel (and its tree shape) has changed. Only dispatch when
  // the node is currently a split whose stored size actually changed, so mounts
  // and panel switches never send invalid or no-op resizes to the server.
  const handleResize = useCallback(
    (key: number, size: number) => {
      const node = panel.walkPath(store.panels.get(panelKey)?.root, key);
      if (node?.split == null || node.split.size === size) return;
      dispatch({ key: panelKey, actions: [panel.resizeSplit({ split: key, size })] });
    },
    [dispatch, panelKey, store],
  );

  const handleClose = useCallback(
    (tabKey: string) => {
      dispatch({ key: panelKey, actions: [panel.removeTab({ key: tabKey })] });
    },
    [dispatch, panelKey],
  );

  // handleCreate maps Base.Mosaic's create gesture onto panel actions. The
  // location decides the structural shape:
  //   - "center": add to the existing leaf (no split).
  //   - an edge (left/right/top/bottom): split the leaf and place the new
  //     content on that side.
  // A bare "+" (no dropped content) inserts a single resourceless tab, a real,
  // stored tab that renders the consumer's selector until SetTabResource fills it.
  // Tabs append to the target leaf (insertTab without an index), and creating a
  // tab selects it through the same onSelect seam as a click, so the consumer's
  // session cursor lands on the new tab.
  const handleCreate = useCallback(
    (key: number, loc: location.Location, tabKeys?: string[]) => {
      const dropped = (tabKeys ?? []).flatMap((raw) => {
        const parsed = ontology.idZ.safeParse(raw);
        return parsed.success ? [parsed.data] : [];
      });
      const actions: panel.Action[] = [];
      // For an edge split the new sibling lands on the child slot that matches
      // the drop side: first (2k) for left/top, last (2k+1) otherwise.
      let targetLeaf = key;
      if (loc !== "center") {
        actions.push(panel.splitLeaf({ leaf: key, location: loc, size: 0.5 }));
        const side = loc === "left" || loc === "top" ? "first" : "last";
        targetLeaf = panel.childPath(key, side);
      }
      const tabs: panel.Tab[] =
        dropped.length === 0
          ? [{ key: uuid.create() }]
          : dropped.map((resource) => ({ key: uuid.create(), resource }));
      for (const tab of tabs) actions.push(panel.insertTab({ tab, targetLeaf }));
      dispatch({ key: panelKey, actions });
      onSelect?.(tabs[tabs.length - 1].key);
    },
    [dispatch, panelKey, onSelect],
  );

  const [portalRef, portalNodes] = Base.usePortal({
    root,
    onSelect: handleSelect,
    children: ({ tabKey, visible }) => (
      <Content panelKey={panelKey} tabKey={tabKey} visible={visible !== false}>
        {children}
      </Content>
    ),
  });

  const tabNameContext = useMemo<TabNameContextValue>(
    () => ({ panelKey, tabName }),
    [panelKey, tabName],
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
        activeTab={activeTab}
        onSelect={handleSelect}
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
