import React, { useState } from "react";
import { Resize } from "../../Atoms";
import { Tabs } from "../../Atoms/Tabs";
import { TabEntry, TabProps, TabsProps } from "../../Atoms/Tabs/Tabs";
import MosaicTree, { MosaicNode } from "./MosaicTree";
import { Location } from "../../util/spatial";
import "./Mosaic.css";

export interface MosaicProps extends Omit<TabsProps, "onDrop" | "tabs"> {
  onDrop: (key: number, tabKey: string, loc: Location) => void;
  onResize: (key: number, size: number) => void;
  tree: MosaicNode;
}

const Mosaic = (props: MosaicProps) => {
  const {
    tree: { tabs, direction, first, last, key, size },
    onResize,
  } = props;
  if (tabs && tabs.length > 0) return <MosaicTabNode {...props} />;
  if (!direction || !last || !first)
    throw new Error("[BUG] - Invalid MosaicTreeNode");

  const _onResize = (sizes: number[]) => onResize(key, sizes[0]);

  return (
    <Resize.Multiple
      direction={direction}
      style={{ position: "relative", height: "100%", width: "100%" }}
      onResize={_onResize}
      initialSizes={size ? [size] : undefined}
    >
      <Mosaic {...props} tree={first} />
      <Mosaic {...props} tree={last} />
    </Resize.Multiple>
  );
};

const MosaicTabNode = ({ tree: node, onDrop, ...props }: MosaicProps) => {
  const { key, tabs } = node as Omit<MosaicNode, "tabs"> & { tabs: TabEntry[] };

  const [dragMask, setDragMask] = useState<Location | null>(null);
  const [currentlyDragging, setCurrentlyDragging] = useState<string | null>(
    null
  );

  const _onDrop = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    const validDrop =
      tabs.filter((t) => t.tabKey !== currentlyDragging).length > 0;
    if (!validDrop) return;
    onDrop(
      key,
      e.dataTransfer.getData("tabKey"),
      insertLocation(getDragLocationPercents(e))
    );
    if (dragMask) setDragMask(null);
    if (currentlyDragging) setCurrentlyDragging(null);
  };

  const onDragOver = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    const loc = insertLocation(getDragLocationPercents(e));
    // get the tab data, get a boolean value checking whether the length of the tabs
    // in node would be zero if the tab was removed
    const validDrop =
      tabs.filter((t) => t.tabKey !== currentlyDragging).length > 0;
    if (loc !== dragMask && validDrop) setDragMask(loc);
  };

  const onDragLeave = () => dragMask && setDragMask(null);

  const onDragEnter = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
  };

  return (
    <>
      <Tabs
        style={{ height: "100%" }}
        tabs={tabs as TabProps[]}
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
      />
      {dragMask && (
        <div
          className="pluto-mosaic__drag-mask"
          style={dragMaskStyle[dragMask]}
        ></div>
      )}
    </>
  );
};

const dragMaskStyle: Record<
  Location,
  { left: string; top: string; width: string; height: string }
> = {
  ["top"]: { left: "0%", top: "0%", width: "100%", height: "50%" },
  ["bottom"]: { left: "0%", top: "50%", width: "100%", height: "50%" },
  ["left"]: { left: "0%", top: "0%", width: "50%", height: "100%" },
  ["right"]: { left: "50%", top: "0%", width: "50%", height: "100%" },
  ["center"]: { left: "0%", top: "0%", width: "100%", height: "100%" },
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

const insertLocation = ({ px, py }: { px: number; py: number }) => {
  if (px > 0.33 && px < 0.66 && py > 0.33 && py < 0.66) return "center";
  const [aY, bY] = [crossHairA(px), crossHairB(px)];
  if (py > aY && py > bY) return "bottom";
  if (py < aY && py < bY) return "top";
  if (py > aY && py < bY) return "left";
  if (py < aY && py > bY) return "right";
  throw new Error("[bug] - invalid insert position");
};

export default Mosaic;
