// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import React, { useState, memo, useCallback } from "react";

import { MosaicLeaf } from "./types";

import { Resize } from "@/core/Resize";
import { Tab, Tabs, TabsProps } from "@/core/Tabs";
import { Location } from "@/spatial";

import "./Mosaic.css";

export interface MosaicProps extends Omit<TabsProps, "onDrop" | "tabs" | "onResize"> {
  onDrop: (key: number, tabKey: string, loc: Location) => void;
  onResize: (key: number, size: number) => void;
  root: MosaicLeaf;
}

export const Mosaic = memo((props: MosaicProps): JSX.Element | null => {
  const { onResize, ...tabsProps } = props;
  const {
    root: { tabs, direction, first, last, key, size },
    emptyContent,
    ...childProps
  } = tabsProps;

  if (tabs !== undefined)
    return <MosaicTabLeaf emptyContent={emptyContent} {...tabsProps} />;

  const _onResize = useCallback(
    (sizes: number[]): void => onResize(key, sizes[0]),
    [onResize]
  );

  if (first == null || last == null) {
    console.warn("Mosaic tree is malformed");
    return null;
  }

  const { props: resizeProps } = Resize.useMultiple({
    direction,
    onResize: _onResize,
    count: 2,
    initialSizes: size != null ? [size] : undefined,
  });

  return (
    <Resize.Multiple align="stretch" className="pluto-mosaic__resize" {...resizeProps}>
      <Mosaic key={first.key} {...childProps} root={first} onResize={onResize} />
      <Mosaic key={last.key} {...childProps} root={last} onResize={onResize} />
    </Resize.Multiple>
  );
});
Mosaic.displayName = "Mosaic";

export interface MosicaTabLeafProps extends Omit<MosaicProps, "onResize"> {}

const MosaicTabLeaf = memo(
  ({ root: node, onDrop, ...props }: MosicaTabLeafProps): JSX.Element => {
    const { key, tabs } = node as Omit<MosaicLeaf, "tabs"> & { tabs: Tab[] };

    const [dragMask, setDragMask] = useState<Location | null>(null);
    const [currentlyDragging, setCurrentlyDragging] = useState<string | null>(null);

    const _onDrop = (e: React.DragEvent<HTMLDivElement>): void => {
      e.preventDefault();
      const validDrop = tabs.filter((t) => t.tabKey !== currentlyDragging).length > 0;
      if (currentlyDragging != null) setCurrentlyDragging(null);
      if (dragMask != null) setDragMask(null);
      if (!validDrop) return;
      onDrop(
        key,
        e.dataTransfer.getData("tabKey"),
        insertLocation(getDragLocationPercents(e))
      );
    };

    const onDragOver = (e: React.DragEvent<HTMLDivElement>): void => {
      e.preventDefault();
      e.stopPropagation();
      const loc = insertLocation(getDragLocationPercents(e));
      // get the tab data, get a boolean value checking whether the length of the tabs
      // in node would be zero if the tab was removed
      const validDrop = tabs.filter((t) => t.tabKey !== currentlyDragging).length > 0;
      if (loc !== dragMask && validDrop) setDragMask(loc);
    };

    const onDragLeave = (): void => {
      if (dragMask != null) setDragMask(null);
    };

    const onDragEnter = (e: React.DragEvent<HTMLDivElement>): void =>
      e.preventDefault();

    return (
      <div style={{ position: "relative", height: "100%" }}>
        <Tabs
          style={{ height: "100%" }}
          tabs={tabs}
          {...props}
          onDrop={_onDrop}
          onDragOver={onDragOver}
          onDragLeave={onDragLeave}
          onDragEnter={onDragEnter}
          selected={node.selected}
          onTabDragStart={(e, tabEntry) => {
            e.dataTransfer.setData("tabKey", tabEntry.tabKey);
            setCurrentlyDragging(tabEntry.tabKey);
          }}
          onTabDragEnd={() => setCurrentlyDragging(null)}
        />
        {dragMask != null && (
          <div
            className="pluto-mosaic__drag-mask"
            style={dragMaskStyle[dragMask]}
          ></div>
        )}
      </div>
    );
  }
);

MosaicTabLeaf.displayName = "MosaicTabLeaf";

const dragMaskStyle: Record<
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
