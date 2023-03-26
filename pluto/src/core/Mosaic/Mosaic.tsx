// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import React, { useState, memo, useCallback } from "react";

import { Location } from "@synnaxlabs/x";

import { MosaicNode } from "./types";

import { Resize } from "@/core/Resize";
import { Tab, Tabs, TabsProps } from "@/core/Tabs";
import { CSS } from "@/css";
import { preventDefault } from "@/util/event";

import "./Mosaic.css";

/** Props for the {@link Mosaic} component */
export interface MosaicProps
  extends Omit<TabsProps, "onDrop" | "tabs" | "onResize" | "onCreate"> {
  root: MosaicNode;
  onDrop: (key: number, tabKey: string, loc: Location) => void;
  onResize: (key: number, size: number) => void;
  onCreate?: (key: number) => void;
}

export const Mosaic = memo((props: MosaicProps): JSX.Element | null => {
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

  if (tabs !== undefined)
    return <MosaicTabLeaf emptyContent={emptyContent} {...tabsProps} />;

  if (first == null || last == null) {
    console.warn("Mosaic tree is malformed");
    return null;
  }

  return (
    <Resize.Multiple
      align="stretch"
      className={CSS.BE("mosaic", "resize")}
      {...resizeProps}
    >
      <Mosaic key={first.key} {...childProps} root={first} onResize={onResize} />
      <Mosaic key={last.key} {...childProps} root={last} onResize={onResize} />
    </Resize.Multiple>
  );
});
Mosaic.displayName = "Mosaic";

interface MosaicTabLeafProps extends Omit<MosaicProps, "onResize"> {}

/** Checks whether the tab can actually be dropped in this location or not */
const validDrop = (tabs: Tab[], currentlyDragging: string | null): boolean =>
  tabs.filter((t) => t.tabKey !== currentlyDragging).length > 0;

const MosaicTabLeaf = memo(
  ({ root: node, onDrop, onCreate, ...props }: MosaicTabLeafProps): JSX.Element => {
    const { key, tabs } = node as Omit<MosaicNode, "tabs"> & { tabs: Tab[] };

    const [dragMask, setDragMask] = useState<Location | null>(null);
    const [currentlyDragging, setCurrentlyDragging] = useState<string | null>(null);

    const handleDrop = (e: React.DragEvent<HTMLDivElement>): void => {
      e.preventDefault();
      setCurrentlyDragging(null);
      setDragMask(null);
      if (!validDrop(tabs, currentlyDragging)) return;
      onDrop(
        key,
        e.dataTransfer.getData("tabKey"),
        insertLocation(getDragLocationPercents(e))
      );
    };

    const handleDragOver = (e: React.DragEvent<HTMLDivElement>): void => {
      e.preventDefault();
      const loc = insertLocation(getDragLocationPercents(e));
      // get the tab data, get a boolean value checking whether the length of the tabs
      // in node would be zero if the tab was removed.
      if (loc !== dragMask && validDrop(tabs, currentlyDragging)) setDragMask(loc);
    };

    const handleDragLeave = (): void => setDragMask(null);

    const handleDragStart = (
      e: React.DragEvent<HTMLDivElement>,
      { tabKey }: Tab
    ): void => {
      e.dataTransfer.setData("tabKey", tabKey);
      setCurrentlyDragging(tabKey);
    };

    const handleDragEnd = (): void => setCurrentlyDragging(null);

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
          <div className={CSS.BE("mosaic", "mask")} style={maskStyle[dragMask]} />
        )}
      </div>
    );
  }
);

MosaicTabLeaf.displayName = "MosaicTabLeaf";

const maskStyle: Record<
  Location,
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
  if (px > 0.33 && px < 0.66 && py > 0.33 && py < 0.66) return "center";
  const [aY, bY] = [crossHairA(px), crossHairB(px)];
  if (py > aY && py > bY) return "bottom";
  if (py < aY && py < bY) return "top";
  if (py > aY && py < bY) return "left";
  if (py < aY && py > bY) return "right";
  throw new Error("[bug] - invalid insert position");
};
