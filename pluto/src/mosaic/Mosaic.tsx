// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/mosaic/Mosaic.css";

import { type box, type location } from "@synnaxlabs/x";
import {
  DragEvent,
  memo,
  MutableRefObject,
  type ReactElement,
  useCallback,
  useRef,
  useState,
} from "react";

import { CSS } from "@/css";
import { Haul } from "@/haul";
import { mapNodes } from "@/mosaic/tree";
import { type Node } from "@/mosaic/types";
import { Portal } from "@/portal";
import { Resize } from "@/resize";
import { Tabs } from "@/tabs";

/** Props for the {@link Mosaic} component */
export interface MosaicProps
  extends Omit<
    Tabs.TabsProps,
    "onDrop" | "tabs" | "onResize" | "onCreate" | "children" | "onDragOver"
  > {
  root: Node;
  onDrop: (key: number, tabKey: string, loc: location.Location) => void;
  onResize: (key: number, size: number) => void;
  onCreate?: (key: number, loc: location.Location, tabKeys?: string[]) => void;
  onFileDrop?: (key: number, loc: location.Location, event: DragEvent) => void;
  children: Tabs.RenderProp;
  activeTab?: string;
}

export const Mosaic = memo((props: MosaicProps): ReactElement => {
  return <MosaicInternal {...props} />;
});

interface MosaicInternalProps extends MosaicProps {}

/***
 * Mosaic renders a tree of tab panes, with the ability to drag and drop tabs to
 * different locations in the tree as well as resize the panes (think of your typical
 * code editor). This component should be used in conjunction with the Mosaic.use hook
 * to implement the mosaic logic and maintain the state.
 *
 * @param props - The props for the Mosaic component. All props not listed below are
 * passed to the Tabs component of each set of tabs in the mosaic.
 * @param props.root - The root of the mosaic tree. This prop is provided by the
 *  Mosaic.use hook.
 * @param props.onDrop - The callback executed when a tab is dropped in a new location.
 * This prop is provided by the Mosaic.use hook.
 * @param props.onResize - The callback executed when a pane is resized. This prop is
 *  provided by the Mosaic.use hook.
 */
const MosaicInternal = memo((props: MosaicInternalProps): ReactElement | null => {
  const { onResize, ...tabsProps } = props;
  const {
    root: { tabs, direction, first, last, key, size },
    emptyContent,
    ...childProps
  } = tabsProps;

  const handleResize = useCallback(
    ([size]: number[]) => onResize(key, size),
    [onResize],
  );

  const { props: resizeProps } = Resize.useMultiple({
    direction,
    onResize: handleResize,
    count: 2,
    initialSizes: size != null ? [size] : undefined,
  });

  let content: ReactElement | null;
  if (tabs !== undefined)
    content = <TabLeaf emptyContent={emptyContent} {...tabsProps} />;
  else if (first != null && last != null)
    content = (
      <Resize.Multiple
        id={`mosaic-${key}`}
        align="stretch"
        className={CSS.BE("mosaic", "resize")}
        {...resizeProps}
      >
        <MosaicInternal
          key={first.key}
          {...childProps}
          root={first}
          onResize={onResize}
        />
        <MosaicInternal
          key={last.key}
          {...childProps}
          root={last}
          onResize={onResize}
        />
      </Resize.Multiple>
    );
  else {
    content = null;
    console.warn("Mosaic tree is malformed");
  }

  return key === 1 ? <Haul.Provider>{content}</Haul.Provider> : content;
});
Mosaic.displayName = "Mosaic";
MosaicInternal.displayName = "Mosaic";

interface TabLeafProps extends Omit<MosaicInternalProps, "onResize"> {}

/**
 * This type should be used when the user wants to drop a tab in the mosaic.
 * Dropping an item with this signature will call the {@link Mosaic} onDrop handler.
 */
export const HAUL_DROP_TYPE = "pluto-mosaic-tab-drop";
/** This type should be used when the user wants to create a new tab in the mosaic.
Dropping an item with this signature will call the {@link Mosaic} onCreate handler. */
export const HAUL_CREATE_TYPE = "pluto-mosaic-tab-create";

/** Checks whether the tab can actually be dropped in this location or not */
const validDrop = (
  tabs: Tabs.Tab[],
  dragging: Haul.Item[],
  hasFileDrop?: boolean,
): boolean => {
  const hasFiles = Haul.filterByType(Haul.FILE_TYPE, dragging).length > 0;
  if (hasFiles && hasFileDrop) return true;
  const drop = Haul.filterByType(HAUL_DROP_TYPE, dragging).map((t) => t.key);
  const willHaveTabRemaining = tabs.filter((t) => !drop.includes(t.tabKey)).length > 0;
  const create = Haul.filterByType(HAUL_CREATE_TYPE, dragging);
  return (
    create.length > 0 ||
    (drop.length > 0 && (willHaveTabRemaining || tabs.length === 0))
  );
};

const TabLeaf = memo(
  ({
    root: node,
    onDrop,
    onCreate,
    activeTab,
    children,
    onFileDrop,
    ...props
  }: TabLeafProps): ReactElement => {
    const { key, tabs } = node as Omit<Node, "tabs"> & { tabs: Tabs.Tab[] };

    const [dragMask, setDragMask] = useState<location.Location | null>(null);

    const hasFileDrop = onFileDrop != null;
    const canDrop: Haul.CanDrop = useCallback(
      ({ items }) => validDrop(tabs, items, hasFileDrop),
      [tabs, hasFileDrop],
    );

    const handleDrop = useCallback(
      ({ items, event }: Haul.OnDropProps): Haul.Item[] => {
        if (event == null) return [];
        setDragMask(null);
        const hasFiles = Haul.filterByType(Haul.FILE_TYPE, items).length > 0;
        const loc =
          tabs.length === 0 ? "center" : insertLocation(getDragLocationPercents(event));
        if (hasFiles) {
          onFileDrop?.(key, loc, event);
          return items;
        }
        const dropped = Haul.filterByType(HAUL_DROP_TYPE, items);
        if (dropped.length > 0) {
          const tabKey = dropped.map(({ key }) => key)[0];
          onDrop(key, tabKey as string, loc);
        }
        const created = Haul.filterByType(HAUL_CREATE_TYPE, items);
        if (created.length > 0) {
          const tabKey = created.map(({ key }) => key);
          onCreate?.(key, loc, tabKey as string[]);
        }
        return dropped;
      },
      [onDrop, tabs.length],
    );

    const handleDragOver = useCallback(
      ({ event }: Haul.OnDragOverProps): void => {
        if (event == null) return;
        const location: location.Location =
          tabs.length === 0 ? "center" : insertLocation(getDragLocationPercents(event));
        setDragMask(location);
      },
      [tabs.length],
    );

    const { startDrag, ...haulProps } = Haul.useDragAndDrop({
      type: "Mosaic",
      canDrop,
      onDrop: handleDrop,
      onDragOver: handleDragOver,
    });

    const dragging = canDrop(Haul.useDraggingState());

    const handleDragLeave = useCallback((): void => setDragMask(null), []);

    const handleDragStart = useCallback(
      (e: DragEvent, { tabKey }: Tabs.Tab): void => {
        startDrag([
          { key: tabKey, type: HAUL_DROP_TYPE, elementID: e.currentTarget.id },
        ]);
      },
      [startDrag],
    );

    const handleTabCreate = useCallback((): void => onCreate?.(key, "center"), [key]);

    return (
      <div id={`mosaic-${key}`} className={CSS.BE("mosaic", "leaf")}>
        <Tabs.Tabs
          id={`tab-${key}`}
          tabs={tabs}
          onDragLeave={handleDragLeave}
          selected={node.selected}
          selectedAltColor={activeTab === node.selected}
          onDragStart={handleDragStart}
          onCreate={handleTabCreate}
          {...props}
          {...haulProps}
        >
          {node.selected != null &&
            children(tabs.find((t) => t.tabKey === node.selected) as Tabs.Spec)}
          {dragging && (
            <div
              style={{
                zIndex: 1000,
                position: "absolute",
                top: 0,
                left: 0,
                width: "100%",
                height: "100%",
              }}
            />
          )}
        </Tabs.Tabs>

        {dragMask != null && (
          <div className={CSS.BE("mosaic", "mask")} style={maskStyle[dragMask]} />
        )}
      </div>
    );
  },
);

TabLeaf.displayName = "MosaicTabLeaf";

const maskStyle: Record<location.Location, box.CSS> = {
  top: { left: "0%", top: "0%", width: "100%", height: "50%" },
  bottom: { left: "0%", top: "50%", width: "100%", height: "50%" },
  left: { left: "0%", top: "0%", width: "50%", height: "100%" },
  right: { left: "50%", top: "0%", width: "50%", height: "100%" },
  center: { left: "0%", top: "0%", width: "100%", height: "100%" },
};

const getDragLocationPercents = (
  e: React.DragEvent<Element>,
): { px: number; py: number } => {
  const rect = e.currentTarget.getBoundingClientRect();
  const x = e.clientX - rect.left;
  const y = e.clientY - rect.top;
  // This means we're in the selector.
  // TODO: This size depends on the theme and the size of the tabs,
  // we need to handle this better in the future.
  if (y < 24) return { px: 0.5, py: 0.5 };
  return { px: x / rect.width, py: y / rect.height };
};

const crossHairA = (px: number): number => px;

const crossHairB = (px: number): number => 1 - px;

const insertLocation = ({ px, py }: { px: number; py: number }): location.Location => {
  if (px > 0.33 && px < 0.66 && py > 0.33 && py < 0.66) return "center";
  const [aY, bY] = [crossHairA(px), crossHairB(px)];
  if (py > aY && py > bY) return "bottom";
  if (py < aY && py < bY) return "top";
  if (py > aY && py < bY) return "left";
  if (py < aY && py > bY) return "right";
  throw new Error("[bug] - invalid insert position");
};

export interface UsePortalProps
  extends Pick<MosaicProps, "root" | "onSelect" | "children"> {}

export type UsePortalReturn = [
  MutableRefObject<Map<string, Portal.Node>>,
  ReactElement[],
];

export const usePortal_ = ({
  root,
  onSelect,
  children,
}: UsePortalProps): UsePortalReturn => {
  const ref = useRef<Map<string, Portal.Node>>(new Map());
  const existing = new Set<string>();
  const portaledNodes = mapNodes(root, (node) => {
    if (node.selected == null) return null;
    let pNode: Portal.Node | undefined = ref.current.get(node.selected);
    const tab = node.tabs?.find((t) => t.tabKey === node.selected);
    if (tab == null) return null;
    if (pNode == null) {
      pNode = new Portal.Node({
        style: "width: 100%; height: 100%; position: relative;",
      });
      // Events don't propagate upward from the portaled node, so we need to bind
      // the onSelect handler here.
      pNode.el.addEventListener("click", () => onSelect?.(tab.tabKey));
      ref.current.set(node.selected, pNode);
    }
    existing.add(node.selected);
    return (
      <Portal.In key={tab.tabKey} node={pNode}>
        {children(tab)}
      </Portal.In>
    );
  }).filter((v) => v != null) as ReactElement[];
  // ref.current.forEach((_, key) => !existing.has(key) && ref.current.delete(key));
  return [ref, portaledNodes];
};

export const usePortal = ({
  root,
  onSelect,
  children,
}: UsePortalProps): UsePortalReturn => {
  const ref = useRef<Map<string, Portal.Node>>(new Map());
  const existing = new Set<string>();
  const portaledNodes = mapNodes(root, (node) =>
    node.tabs?.map((tab) => {
      let pNode: Portal.Node | undefined = ref.current.get(tab.tabKey);
      if (tab == null) return null;
      if (pNode == null) {
        pNode = new Portal.Node({
          style: "width: 100%; height: 100%; position: relative;",
        });
        // Events don't propagate upward from the portaled node, so we need to bind
        // the onSelect handler here.
        pNode.el.addEventListener("click", () => onSelect?.(tab.tabKey));
        ref.current.set(tab.tabKey, pNode);
      }
      existing.add(tab.tabKey);
      return (
        <Portal.In key={tab.tabKey} node={pNode}>
          {children({ ...tab, visible: tab.tabKey === node.selected })}
        </Portal.In>
      );
    }),
  )
    .flat()
    .filter((v) => v != null) as ReactElement[];
  // ref.current.forEach((_, key) => !existing.has(key) && ref.current.delete(key));
  return [ref, portaledNodes];
};
