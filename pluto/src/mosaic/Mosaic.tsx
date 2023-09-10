// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState, memo, useCallback, type ReactElement } from "react";


import { CSS } from "@/css";
import { Haul } from "@/haul";
import { type CanDrop } from "@/haul/Haul";
import { type Node } from "@/mosaic/types";
import { Resize } from "@/resize";
import { Tabs } from "@/tabs";

import "@/mosaic/Mosaic.css";
import { box, location } from "@synnaxlabs/x";

/** Props for the {@link Mosaic} component */
export interface MosaicProps
  extends Omit<Tabs.TabsProps, "onDrop" | "tabs" | "onResize" | "onCreate"> {
  root: Node;
  onDrop: (key: number, tabKey: string, loc: location.Location) => void;
  onResize: (key: number, size: number) => void;
  onCreate?: (key: number) => void;
}

/***
 * Mosaic renders a tree of tab panes, with the ability to drag and drop tabs to
 * different locations in the tree as well as resize the panes (think of your typical
 * code editor). This component should be used in conjuction with the Mosaic.use hook
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
export const Mosaic = memo((props: MosaicProps): ReactElement | null => {
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
        align="stretch"
        className={CSS.BE("mosaic", "resize")}
        {...resizeProps}
      >
        <Mosaic key={first.key} {...childProps} root={first} onResize={onResize} />
        <Mosaic key={last.key} {...childProps} root={last} onResize={onResize} />
      </Resize.Multiple>
    );
  else {
    content = null;
    console.warn("Mosaic tree is malformed");
  }

  return key === 1 ? <Haul.Provider>{content}</Haul.Provider> : content;
});
Mosaic.displayName = "Mosaic";

interface TabLeafProps extends Omit<MosaicProps, "onResize"> {}

export const HAUL_TYPE = "pluto-mosaic-tab";

/** Checks whether the tab can actually be dropped in this location or not */
const validDrop = (tabs: Tabs.Tab[], dragging: Haul.Item[]): boolean => {
  const keys = dragging.filter(({ type }) => type === HAUL_TYPE).map((t) => t.key);
  const willHaveTabRemaining = tabs.filter((t) => !keys.includes(t.tabKey)).length > 0;
  return keys.length > 0 && (willHaveTabRemaining || tabs.length === 0);
};

const TabLeaf = memo(
  ({ root: node, onDrop, onCreate, ...props }: TabLeafProps): ReactElement => {
    const { key, tabs } = node as Omit<Node, "tabs"> & { tabs: Tabs.Tab[] };

    const [dragMask, setDragMask] = useState<location.Location | null>(null);

    const canDrop: CanDrop = useCallback(({ items }) => validDrop(tabs, items), [tabs]);

    const handleDrop = useCallback(
      ({ items, event }: Haul.OnDropProps): Haul.Item[] => {
        setDragMask(null);
        const dropped = Haul.filterByType(HAUL_TYPE, items);
        const tabKey = dropped.map(({ key }) => key)[0];
        const location: location.Location =
          tabs.length === 0
            ? "center"
            : insertLocation(getDragLocationPercents(event));
        onDrop(key, tabKey as string, location);
        return dropped;
      },
      [onDrop, tabs.length],
    );

    const handleDragOver = useCallback(
      ({ event }: Haul.OnDragOverProps): void => {
        const location: location.Location =
          tabs.length === 0
            ? "center"
            : insertLocation(getDragLocationPercents(event));
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

    const handleDragLeave = useCallback((): void => setDragMask(null), []);

    const handleDragStart = useCallback(
      (_: unknown, { tabKey }: Tabs.Tab): void =>
        startDrag([{ key: tabKey, type: HAUL_TYPE }]),
      [startDrag],
    );

    const handleCreate = useCallback((): void => onCreate?.(key), [key]);

    return (
      <div className={CSS.BE("mosaic", "leaf")}>
        <Tabs.Tabs
          tabs={tabs}
          onDragLeave={handleDragLeave}
          selected={node.selected}
          onDragStart={handleDragStart}
          onCreate={handleCreate}
          {...props}
          {...haulProps}
        />
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
