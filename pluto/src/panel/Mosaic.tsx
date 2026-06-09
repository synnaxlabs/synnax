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
import { type Icon } from "@/icon";
import { Mosaic as Base } from "@/mosaic";
import { useDispatch, useRetrieve } from "@/panel/queries";
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
  // onSelect fires when the operator clicks a tab. The panel-aware shell does
  // not persist this — consumers route it into their session-state store.
  onSelect?: (tabKey: string) => void;
  // children renders a tab's content from its resolved content union.
  children: (props: MosaicTabRenderProps) => ReactElement | null;
  // tabName renders a tab's display name from its content union. When omitted, tab
  // strips show the default (empty) name. Rename, when supported, is the consumer's
  // to wire through the underlying content (e.g. renaming the resource), so the
  // panel layer needs no rename prop of its own.
  tabName?: (props: MosaicTabNameProps) => ReactElement | null;
  // tabInfo resolves a tab's passive display attributes (icon, unsaved-changes
  // indicator) from its content union. Unlike tabName it returns data baked onto the
  // tab spec, not a rendered node.
  tabInfo?: (props: {
    tabKey: string;
    resource: ontology.ID | null;
    view: panel.TabView | null;
  }) => { icon?: Icon.ReactElement | string; unsavedChanges?: boolean };
}

// TabContent is a tab's resolved content union. At most one of resource or view is
// non-null; both null means the tab has no content yet.
export interface TabContent {
  resource: ontology.ID | null;
  view: panel.TabView | null;
}

// adaptToMosaic walks the typed panel tree and produces the Base.Node shape
// with path-derived numeric keys (root = 1, first child = 2k, last child =
// 2k + 1). It also builds a tab → content map so the render props can
// resolve a tab's content without re-traversing the tree.
interface AdaptResult {
  root: Base.Node;
  // Maps every tab key to its content union. A key absent from the map is not a
  // tab in this panel.
  contents: Map<string, TabContent>;
}

const adaptToMosaic = (
  root: panel.Node | undefined,
  activeTab: string | undefined,
  tabInfo?: MosaicProps["tabInfo"],
): AdaptResult => {
  const contents = new Map<string, TabContent>();
  const visit = (node: panel.Node | undefined, key: number): Base.Node => {
    if (node == null) return { key };
    if (node.split != null)
      return {
        key,
        direction: node.split.direction,
        size: node.split.size,
        first: visit(node.split.first, key * 2),
        last: visit(node.split.last, key * 2 + 1),
      };

    if (node.leaf == null) return { key, tabs: [] };
    const tabs: Tabs.Tab[] = node.leaf.tabs.map((t) => {
      const resource = t.resource ?? null;
      const view = t.view ?? null;
      contents.set(t.key, { resource, view });
      const info = tabInfo?.({ tabKey: t.key, resource, view }) ?? {};
      // Resource and view tabs carry a renameable name; empty (selector) tabs do not.
      return {
        tabKey: t.key,
        name: "",
        closable: true,
        editable: resource != null || view != null,
        icon: info.icon,
        unsavedChanges: info.unsavedChanges,
      };
    });
    const selected =
      activeTab != null && tabs.some((t) => t.tabKey === activeTab)
        ? activeTab
        : tabs[0]?.tabKey;
    return { key, tabs, selected };
  };
  return { root: visit(root, 1), contents };
};

// TabNameContext carries the per-tab content map and the consumer's tabName
// resolver down to the Base.Mosaic tab strip, which only exposes Tabs.NameProps.
// A module-stable Name component reads it so tab names never remount on tree
// changes.
interface TabNameContextValue {
  contents: Map<string, TabContent>;
  tabName?: (props: MosaicTabNameProps) => ReactElement | null;
}

const [TabNameContext, useTabNameContext] = context.create<TabNameContextValue>({
  displayName: "Panel.Mosaic.TabName",
  defaultValue: { contents: new Map() },
});

const TabName: ComponentType<Tabs.NameProps> = (props) => {
  const { contents, tabName } = useTabNameContext();
  if (tabName == null) return <Tabs.DefaultName {...props} />;
  const content = contents.get(props.tabKey);
  return tabName({
    ...props,
    resource: content?.resource ?? null,
    view: content?.view ?? null,
  });
};

// Mosaic renders a Flux-backed Panel as a Base.Mosaic. The architecture is a
// direct parallel of the legacy console layout mosaic; the only material
// difference is that gestures dispatch panel actions through the SY-3038
// substrate instead of mutating a Redux slice. Other connected consoles
// observe each mutation through the panel action channel and apply it via
// the panel reducer, so cross-client sync is automatic.
//
// Tab content is delivered through the children render prop, which receives the
// tabKey and its resolved content union. The portal pattern borrowed from
// Base.Mosaic preserves content lifetime across structural changes — moving a tab
// between leaves does not unmount its content.
//
// Tab display names are the consumer's responsibility through the tabName render
// prop, which receives the same content union. The default tab name on the
// underlying Tabs.Tab is empty; consumers resolve a name (and rename) from the
// referenced resource or view.
export const Mosaic = ({
  panelKey,
  activeTab,
  onSelect,
  children,
  tabName,
  tabInfo,
  ...rest
}: MosaicProps): ReactElement | null => {
  const { data: p } = useRetrieve({ key: panelKey });
  const { dispatch } = useDispatch();

  const { root, contents } = useMemo(
    () => adaptToMosaic(p?.root, activeTab, tabInfo),
    [p?.root, activeTab, tabInfo],
  );

  const handleSelect = useCallback((tabKey: string) => onSelect?.(tabKey), [onSelect]);

  const handleDrop = useCallback(
    (key: number, tabKey: string, _loc: location.Location, index?: number) => {
      dispatch({
        key: panelKey,
        actions: [panel.moveTab({ key: tabKey, targetLeaf: key, index: index ?? 0 })],
      });
    },
    [dispatch, panelKey],
  );

  const handleResize = useCallback(
    (key: number, size: number) => {
      dispatch({ key: panelKey, actions: [panel.resizeSplit({ split: key, size })] });
    },
    [dispatch, panelKey],
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
  // A bare "+" (no dropped content) inserts a single resourceless tab — a real,
  // stored tab that renders the consumer's selector until SetTabResource fills it.
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
        targetLeaf = loc === "left" || loc === "top" ? key * 2 : key * 2 + 1;
      }
      if (dropped.length === 0)
        actions.push(
          panel.insertTab({ tab: { key: uuid.create() }, targetLeaf, index: 0 }),
        );
      else
        for (const resource of dropped)
          actions.push(
            panel.insertTab({
              tab: { key: uuid.create(), resource },
              targetLeaf,
              index: 0,
            }),
          );
      dispatch({ key: panelKey, actions });
    },
    [dispatch, panelKey],
  );

  const [portalRef, portalNodes] = Base.usePortal({
    root,
    onSelect: handleSelect,
    children: ({ tabKey, visible }) => {
      const content = contents.get(tabKey);
      if (content == null) return null;
      return children({
        tabKey,
        resource: content.resource,
        view: content.view,
        visible: visible !== false,
      });
    },
  });

  const tabNameContext = useMemo<TabNameContextValue>(
    () => ({ contents, tabName }),
    [contents, tabName],
  );

  const renderProp = useCallback<Tabs.RenderProp>(
    (props) => {
      const node = portalRef.current.get(props.tabKey);
      if (node == null) return null;
      return <Portal.Out node={node} />;
    },
    [portalRef],
  );

  if (p == null) return null;

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
