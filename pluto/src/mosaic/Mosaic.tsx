// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/mosaic/Mosaic.css";

import { type box, type location, type xy } from "@synnaxlabs/x";
import {
  type DragEvent,
  memo,
  type ReactElement,
  type RefObject,
  useCallback,
  useRef,
  useState,
} from "react";

import { CSS } from "@/css";
import { type Flex } from "@/flex";
import { Haul } from "@/haul";
import { mapNodes } from "@/mosaic/tree";
import { type Node } from "@/mosaic/types";
import { Portal } from "@/portal";
import { Resize } from "@/resize";
import { Tabs } from "@/tabs";

/** Props for the {@link Mosaic} component */
export interface MosaicProps
  extends
    Pick<
      Tabs.TabsProps,
      | "onSelect"
      | "contextMenu"
      | "emptyContent"
      | "onRename"
      | "onClose"
      | "addTooltip"
    >,
    Omit<
      Flex.BoxProps,
      "contextMenu" | "onSelect" | "children" | "onResize" | "onDrop"
    > {
  root: Node;
  onDrop: (
    key: number,
    droppedTabKey: string,
    loc: location.Location,
    index?: number,
  ) => void;
  onResize: (key: number, size: number) => void;
  onCreate?: (key: number, loc: location.Location, tabKeys?: string[]) => void;
  onReorder?: (
    key: number,
    droppedTabKey: string,
    targetTabKey: string,
    location: location.X,
  ) => void;
  onFileDrop?: (key: number, loc: location.Location, event: DragEvent) => void;
  children: Tabs.RenderProp;
  activeTab?: string;
}

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
export const Mosaic = memo(
  ({
    root,
    onDrop,
    onResize,
    onCreate,
    onFileDrop,
    children,
    activeTab,
    emptyContent,
    onSelect,
    onClose,
    onRename,
    onReorder,
    contextMenu,
    addTooltip,
    className,
    ...rest
  }: MosaicProps): ReactElement | null => {
    const { tabs, direction, first, last, key, size } = root;
    const childProps = {
      onDrop,
      onResize,
      onCreate,
      onFileDrop,
      children,
      onClose,
      contextMenu,
      onSelect,
      onRename,
      onReorder,
      activeTab,
      addTooltip,
    };

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
    let extraProps: Partial<Flex.BoxProps> = {};
    if (key == 1)
      extraProps = {
        ...rest,
        className: CSS(CSS.B("mosaic"), className),
      };

    let content: ReactElement | null;
    if (tabs !== undefined)
      content = (
        <TabLeaf
          root={root}
          emptyContent={emptyContent}
          {...extraProps}
          {...childProps}
        />
      );
    else if (first != null && last != null)
      content = (
        <Resize.Multiple
          id={`mosaic-${key}`}
          align="stretch"
          {...resizeProps}
          {...extraProps}
        >
          <Mosaic key={first.key} {...childProps} root={first} onResize={onResize} />
          <Mosaic key={last.key} {...childProps} root={last} onResize={onResize} />
        </Resize.Multiple>
      );
    else {
      content = null;
      console.warn("Mosaic tree is malformed");
    }

    return content;
  },
);
Mosaic.displayName = "Mosaic";

interface TabLeafProps extends Omit<
  MosaicProps,
  "onResize" | "onDragStart" | "onDragEnd"
> {}

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
    className,
    onReorder,
    onFileDrop,
    addTooltip,
    ...rest
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
        const index = getDragLocationIndex(event);
        setDragMask(null);
        const hasFiles = Haul.filterByType(Haul.FILE_TYPE, items).length > 0;
        const { percents } = getDragLocationPercents(event);
        const loc = tabs.length === 0 ? "center" : insertLocation(percents);
        if (hasFiles) {
          onFileDrop?.(key, loc, event);
          return items;
        }
        const dropped = Haul.filterByType(HAUL_DROP_TYPE, items);
        if (dropped.length > 0) {
          const tabKey = dropped.map(({ key }) => key)[0];
          onDrop(key, tabKey as string, loc, index);
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
        const { percents, inSelector } = getDragLocationPercents(event);
        let location: location.Location | null = null;
        if (!inSelector)
          location = tabs.length === 0 ? "center" : insertLocation(percents);
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
      (e: DragEvent<HTMLElement>, { tabKey }: Tabs.Tab): void => {
        startDrag([
          { key: tabKey, type: HAUL_DROP_TYPE, elementID: e.currentTarget.id },
        ]);
      },
      [startDrag],
    );

    const handleTabCreate = useCallback(
      (): void => onCreate?.(key, "center"),
      [key, onCreate],
    );

    const isEmpty = key == 1 && tabs.length == 0;

    return (
      <Tabs.Tabs
        id={`tab-${key}`}
        tabs={tabs}
        className={CSS(className, isEmpty && dragMask != null && CSS.M("drag-over"))}
        onDragLeave={handleDragLeave}
        selected={node.selected}
        selectedAltColor={activeTab === node.selected}
        onDragStart={handleDragStart}
        onCreate={onCreate ? handleTabCreate : undefined}
        addTooltip={addTooltip}
        {...haulProps}
        {...rest}
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
        {dragMask != null && (
          <div className={CSS.BE("mosaic", "mask")} style={maskStyle[dragMask]} />
        )}
      </Tabs.Tabs>
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

interface DragLocationPercentsResult {
  percents: xy.XY;
  inSelector: boolean;
}

const getDragLocationPercents = (
  e: React.DragEvent<Element>,
): DragLocationPercentsResult => {
  const rect = e.currentTarget.getBoundingClientRect();
  const x = e.clientX - rect.left;
  const y = e.clientY - rect.top;
  // This means we're in the selector.
  // TODO: This size depends on the theme and the size of the tabs,
  // we need to handle this better in the future.
  if (y < 24) return { percents: { x: 0.5, y: 0.5 }, inSelector: true };
  return { percents: { x: x / rect.width, y: y / rect.height }, inSelector: false };
};

const getDragLocationIndex = (e: React.DragEvent<Element>): number | undefined => {
  const btn = (e.target as HTMLElement).closest(".pluto-tabs-selector__btn");
  if (btn == null) return undefined;
  return Array.from(btn.parentElement?.children ?? []).indexOf(btn);
};

const crossHairA = (px: number): number => px;

const crossHairB = (px: number): number => 1 - px;

const insertLocation = ({ x: px, y: py }: xy.XY): location.Location => {
  if (px > 0.33 && px < 0.66 && py > 0.33 && py < 0.66) return "center";
  const [aY, bY] = [crossHairA(px), crossHairB(px)];
  if (py > aY && py > bY) return "bottom";
  if (py < aY && py < bY) return "top";
  if (py > aY && py < bY) return "left";
  if (py < aY && py > bY) return "right";
  throw new Error("[bug] - invalid insert position");
};

export interface UsePortalProps extends Pick<
  MosaicProps,
  "root" | "onSelect" | "children"
> {}

export type UsePortalReturn = [RefObject<Map<string, Portal.Node>>, ReactElement[]];

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
    .filter((v) => v != null);
  ref.current.forEach((_, key) => !existing.has(key) && ref.current.delete(key));
  return [ref, portaledNodes];
};
