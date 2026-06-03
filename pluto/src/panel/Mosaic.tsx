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
import { type ReactElement, useCallback, useMemo } from "react";

import { Mosaic as Base } from "@/mosaic";
import { useDispatch, useRetrieve } from "@/panel/queries";
import { Portal } from "@/portal";
import { type Tabs } from "@/tabs";

// MosaicTabRenderProps is the contract for the children render prop. The
// component decides how to render the visualization referenced by `resource`;
// the panel layer is intentionally renderer-agnostic so different consumers
// (console, integration tests, future apps) can plug in their own resolution.
export interface MosaicTabRenderProps {
  tabKey: string;
  // resource is null when the tab has not yet been assigned a visualization. The
  // consumer renders its selector for a null resource and swaps it via SetTabResource.
  resource: ontology.ID | null;
  visible: boolean;
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
  | "Name"
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
  // onRenameTab forwards inline rename gestures from the tab strip. Tab display
  // names are derived from the referenced resource, so renaming a tab really
  // means renaming the underlying resource — a per-type operation the panel
  // layer cannot perform on its own.
  onRenameTab?: (tabKey: string, name: string, resource: ontology.ID) => void;
  children: (props: MosaicTabRenderProps) => ReactElement | null;
}

// adaptToMosaic walks the typed panel tree and produces the Base.Node shape
// with path-derived numeric keys (root = 1, first child = 2k, last child =
// 2k + 1). It also builds a tab → resource map so the render prop can
// resolve a tab's reference without re-traversing the tree.
interface AdaptResult {
  root: Base.Node;
  // Maps every tab key to its resource, or null when the tab has no resource yet.
  // A key absent from the map is not a tab in this panel.
  resources: Map<string, ontology.ID | null>;
}

const adaptToMosaic = (
  root: panel.Node | undefined,
  activeTab: string | undefined,
): AdaptResult => {
  const resources = new Map<string, ontology.ID | null>();
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
      resources.set(t.key, t.resource ?? null);
      return { tabKey: t.key, name: "", closable: true, editable: true };
    });
    const selected =
      activeTab != null && tabs.some((t) => t.tabKey === activeTab)
        ? activeTab
        : tabs[0]?.tabKey;
    return { key, tabs, selected };
  };
  return { root: visit(root, 1), resources };
};

// Mosaic renders a Flux-backed Panel as a Base.Mosaic. The architecture is a
// direct parallel of the legacy console layout mosaic; the only material
// difference is that gestures dispatch panel actions through the SY-3038
// substrate instead of mutating a Redux slice. Other connected consoles
// observe each mutation through the panel action channel and apply it via
// the panel reducer, so cross-client sync is automatic.
//
// Tab content is delivered through the children render prop, which receives
// the tabKey and its underlying ontology resource. The portal pattern
// borrowed from Base.Mosaic preserves content lifetime across structural
// changes — moving a tab between leaves does not unmount its content.
//
// Tab display names are the consumer's responsibility through the Name prop,
// which Base.Mosaic invokes per tab. The default tab name on the underlying
// Tabs.Tab is empty; consumers supply a Name component that resolves the
// referenced resource via its Flux store.
export const Mosaic = ({
  panelKey,
  activeTab,
  onSelect,
  onRenameTab,
  children,
  ...rest
}: MosaicProps): ReactElement | null => {
  const { data: p } = useRetrieve({ key: panelKey });
  const { dispatch } = useDispatch();

  const { root, resources } = useMemo(
    () => adaptToMosaic(p?.root, activeTab),
    [p?.root, activeTab],
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

  const handleRename = useCallback(
    (tabKey: string, name: string) => {
      const resource = resources.get(tabKey);
      if (resource == null) return;
      onRenameTab?.(tabKey, name, resource);
    },
    [resources, onRenameTab],
  );

  const [portalRef, portalNodes] = Base.usePortal({
    root,
    onSelect: handleSelect,
    children: ({ tabKey, visible }) => {
      if (!resources.has(tabKey)) return null;
      const resource = resources.get(tabKey) ?? null;
      return children({ tabKey, resource, visible: visible !== false });
    },
  });

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
    <>
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
        onRename={onRenameTab != null ? handleRename : undefined}
      >
        {renderProp}
      </Base.Mosaic>
    </>
  );
};
