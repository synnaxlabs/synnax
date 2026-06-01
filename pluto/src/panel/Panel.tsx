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
  resource: ontology.ID;
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
  resources: Map<string, ontology.ID>;
}

const adaptToMosaic = (
  root: panel.Node | undefined,
  activeTab: string | undefined,
): AdaptResult => {
  const resources = new Map<string, ontology.ID>();
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
      resources.set(t.key, t.resource);
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

// directionAndSideFromLocation maps an edge-drop location to a (direction,
// side) pair for SplitLeaf. A left/right drop splits along the x axis with
// the new sibling on first/last; top/bottom splits along y.
const directionAndSideFromLocation = (
  loc: location.Location,
): { direction: "x" | "y"; side: "first" | "last" } => {
  switch (loc) {
    case "left":
      return { direction: "x", side: "first" };
    case "right":
      return { direction: "x", side: "last" };
    case "top":
      return { direction: "y", side: "first" };
    default:
      return { direction: "y", side: "last" };
  }
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
export const Panel = ({
  panelKey,
  activeTab,
  onSelect,
  onRenameTab,
  children,
  ...rest
}: MosaicProps): ReactElement | null => {
  const { data: doc } = useRetrieve({ key: panelKey });
  const { dispatch } = useDispatch();

  const { root, resources } = useMemo(
    () => adaptToMosaic(doc?.root, activeTab),
    [doc?.root, activeTab],
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

  // handleCreate covers two edge-drop gestures:
  //   1. tabKeys is null: the operator hit the "+" affordance — split the
  //      target leaf and leave the new sibling empty so the consumer can
  //      decide what to put there (typically via a selector UI handled
  //      outside this component).
  //   2. tabKeys is non-null: the operator dragged content (ontology IDs)
  //      from a sidebar onto the leaf edge — split and insert one tab per
  //      parseable ID into the new sibling leaf in a single dispatch.
  const handleCreate = useCallback(
    (key: number, loc: location.Location, tabKeys?: string[]) => {
      const { side } = directionAndSideFromLocation(loc);
      const actions: panel.Action[] = [
        panel.splitLeaf({ leaf: key, location: loc, size: 0.5 }),
      ];
      if (tabKeys != null && tabKeys.length > 0) {
        // After splitLeaf the original leaf moves to the side opposite the
        // new sibling. The new sibling's path-derived key is the child slot
        // that matches `side` on the new parent split, which itself takes
        // the original leaf's path key.
        const newLeafKey = side === "first" ? key * 2 : key * 2 + 1;
        for (const raw of tabKeys) {
          const parsed = ontology.idZ.safeParse(raw);
          if (!parsed.success) continue;
          actions.push(
            panel.insertTab({
              tab: { key: uuid.create(), resource: parsed.data },
              targetLeaf: newLeafKey,
              index: 0,
            }),
          );
        }
      }
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
      const resource = resources.get(tabKey);
      if (resource == null) return null;
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

  if (doc == null) return null;

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
