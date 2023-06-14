// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import React, { useState, memo, useCallback, ReactElement } from "react";

import { Location, CrudeLocation } from "@synnaxlabs/x";

import { CSS } from "@/core/css";
import { Haul, Hauled } from "@/core/haul";
import { MosaicNode } from "@/core/std/Mosaic/types";
import { Resize } from "@/core/std/Resize";
import { Tab, Tabs, TabsProps } from "@/core/std/Tabs";
import { preventDefault } from "@/util/event";

import "@/core/std/Mosaic/Mosaic.css";

/** Props for the {@link Mosaic} component */
export interface MosaicProps
  extends Omit<TabsProps, "onDrop" | "tabs" | "onResize" | "onCreate"> {
  root: MosaicNode;
  onDrop: (key: number, tabKey: string, loc: Location) => void;
  onResize: (key: number, size: number) => void;
  onCreate?: (key: number) => void;
}

export const Mosaic = memo((props: MosaicProps): ReactElement | null => {
  const { onResize, ...tabsProps } = props;
  const {
    root: { tabs, direction, first, last, key, size },
    emptyContent,
    ...childProps
  } = tabsProps;

  const _onResize = useCallback(
    (sizes: number[]): void => onResize(key, sizes[0]),
    [onResize]
  );

  const { props: resizeProps } = Resize.useMultiple({
    direction,
    onResize: _onResize,
    count: 2,
    initialSizes: size != null ? [size] : undefined,
  });

  let content: ReactElement | null;
  if (tabs !== undefined)
    content = <MosaicTabLeaf emptyContent={emptyContent} {...tabsProps} />;
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

interface MosaicTabLeafProps extends Omit<MosaicProps, "onResize"> {}

const DRAGGING_TYPE = "pluto-mosaic-tab";

/** Checks whether the tab can actually be dropped in this location or not */
const validDrop = (tabs: Tab[], dragging: Hauled[]): boolean => {
  const keys = dragging.filter(({ type }) => type === DRAGGING_TYPE).map((t) => t.key);
  return keys.length > 0 && tabs.filter((t) => !keys.includes(t.tabKey)).length > 0;
};

const MosaicTabLeaf = memo(
  ({ root: node, onDrop, onCreate, ...props }: MosaicTabLeafProps): ReactElement => {
    const { key, tabs } = node as Omit<MosaicNode, "tabs"> & { tabs: Tab[] };

    const [dragMask, setDragMask] = useState<Location | null>(null);
    const {
      dragging,
      startDrag: onDragStart,
      endDrag: handleDragEnd,
    } = Haul.useState();

    const handleDrop = (e: React.DragEvent<HTMLDivElement>): void => {
      e.preventDefault();
      setDragMask(null);
      if (!validDrop(tabs, dragging)) return;
      const tabKey = dragging.map(({ key }) => key)[0];
      onDrop(key, tabKey, insertLocation(getDragLocationPercents(e)));
    };

    const handleDragOver = (e: React.DragEvent<HTMLDivElement>): void => {
      e.preventDefault();
      const loc = insertLocation(getDragLocationPercents(e));
      // get the tab data, get a boolean value checking whether the length of the tabs
      // in node would be zero if the tab was removed.
      if (loc !== dragMask && validDrop(tabs, dragging)) setDragMask(loc);
    };

    const handleDragLeave = (): void => setDragMask(null);

    const handleDragStart = (
      _: React.DragEvent<HTMLDivElement>,
      { tabKey }: Tab
    ): void => onDragStart([{ key: tabKey, type: DRAGGING_TYPE }]);

    const handleCreate = (): void => onCreate?.(key);

    return (
      <div className={CSS.BE("mosaic", "leaf")}>
        <Tabs
          tabs={tabs}
          onDrop={handleDrop}
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onDragEnter={preventDefault}
          selected={node.selected}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
          onCreate={handleCreate}
          {...props}
        />
        {dragMask != null && (
          <div className={CSS.BE("mosaic", "mask")} style={maskStyle[dragMask.crude]} />
        )}
      </div>
    );
  }
);

MosaicTabLeaf.displayName = "MosaicTabLeaf";

const maskStyle: Record<
  CrudeLocation,
  { left: string; top: string; width: string; height: string }
> = {
  top: { left: "0%", top: "0%", width: "100%", height: "50%" },
  bottom: { left: "0%", top: "50%", width: "100%", height: "50%" },
  left: { left: "0%", top: "0%", width: "50%", height: "100%" },
  right: { left: "50%", top: "0%", width: "50%", height: "100%" },
  center: { left: "0%", top: "0%", width: "100%", height: "100%" },
};

const getDragLocationPercents = (
  e: React.DragEvent<HTMLDivElement>
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

const insertLocation = ({ px, py }: { px: number; py: number }): Location => {
  if (px > 0.33 && px < 0.66 && py > 0.33 && py < 0.66) return Location.center;
  const [aY, bY] = [crossHairA(px), crossHairB(px)];
  if (py > aY && py > bY) return Location.bottom;
  if (py < aY && py < bY) return Location.top;
  if (py > aY && py < bY) return Location.left;
  if (py < aY && py > bY) return Location.right;
  throw new Error("[bug] - invalid insert position");
};
