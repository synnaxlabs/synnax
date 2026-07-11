// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/mosaic/Mosaic.css";

import { box, type location, xy } from "@synnaxlabs/x";
import {
  type CSSProperties,
  type DragEventHandler,
  type ReactElement,
  useCallback,
  useRef,
  useState,
} from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Haul } from "@/haul";
import { useCombinedRefs } from "@/hooks";
import { useContext } from "@/mosaic/Frame";
import { filterTabCreateHaulItems, filterTabDropHaulItems } from "@/mosaic/haul";
import { Tabs } from "@/tabs";

const MASK_STYLE: Record<location.Location, CSSProperties> = {
  top: { left: "0%", top: "0%", width: "100%", height: "50%" },
  bottom: { left: "0%", top: "50%", width: "100%", height: "50%" },
  left: { left: "0%", top: "0%", width: "50%", height: "100%" },
  right: { left: "50%", top: "0%", width: "50%", height: "100%" },
  center: { left: "0%", top: "0%", width: "100%", height: "100%" },
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

const TAB_KEY_SELECTOR = "[data-tab-key]";
const TAB_LIST_SELECTOR = '[role="tablist"]';

const leafTabKeys = (el: HTMLElement): string[] =>
  Array.from(el.querySelectorAll(TAB_KEY_SELECTOR))
    .map((tab) => tab.getAttribute("data-tab-key"))
    .filter((key): key is string => key != null);

interface Region {
  location: location.Location;
  index?: number;
}

const resolveRegion = (el: HTMLElement, event: React.DragEvent): Region => {
  if (leafTabKeys(el).length === 0) return { location: "center" };
  const cursor = xy.construct(event.clientX, event.clientY);
  const tabList = el.querySelector(TAB_LIST_SELECTOR);
  if (tabList != null && box.contains(box.construct(tabList), cursor))
    return { location: "center", index: Tabs.getInsertionIndex(tabList, cursor) };
  const b = box.construct(el);
  return {
    location: insertLocation({
      x: (cursor.x - box.left(b)) / box.width(b),
      y: (cursor.y - box.top(b)) / box.height(b),
    }),
  };
};

export interface LeafProps extends Omit<Flex.BoxProps, "onDrop" | "onDragOver"> {
  /** The key identifying this leaf, passed to the Frame's drop handlers. */
  nodeKey: number;
}

/**
 * Leaf is the single haul drop target for one pane of a mosaic. It resolves the
 * drop region from the cursor: within the pane's tab strip it inserts at an index
 * (showing the strip's insertion indicator), at an edge it splits (showing a mask
 * over the affected half), and in the middle it drops in the center. Resolved
 * drops are reported to the Frame's onDrop, onCreate, and onFileDrop handlers
 * with this leaf's key.
 *
 * The leaf discovers its tabs and strip from the DOM: tabs are elements with a
 * `data-tab-key` attribute and the strip is the element with `role="tablist"`,
 * both rendered by the Tabs parts composed inside it.
 */
export const Leaf = ({
  nodeKey,
  className,
  children,
  onDragLeave,
  ref,
  ...rest
}: LeafProps): ReactElement => {
  const { onDrop, onCreate, onFileDrop } = useContext("Mosaic.Leaf");
  const internalRef = useRef<HTMLDivElement | null>(null);
  const combinedRef = useCombinedRefs(ref, internalRef);
  const [mask, setMask] = useState<location.Location | null>(null);
  const [insertionIndex, setInsertionIndex] = useState<number | null>(null);

  const hasFileDrop = onFileDrop != null;
  const canDrop: Haul.CanDrop = useCallback(
    ({ items }) => {
      const hasFiles = Haul.filterByType(Haul.FILE_TYPE, items).length > 0;
      if (hasFiles && hasFileDrop) return true;
      if (filterTabCreateHaulItems(items).length > 0) return true;
      const dropped = filterTabDropHaulItems(items).map(({ key }) => key);
      if (dropped.length === 0) return false;
      const el = internalRef.current;
      if (el == null) return false;
      const keys = leafTabKeys(el);
      return keys.length === 0 || keys.some((key) => !dropped.includes(key));
    },
    [hasFileDrop],
  );

  const handleDrop = useCallback(
    ({ items, event }: Haul.OnDropProps): Haul.Item[] => {
      setMask(null);
      setInsertionIndex(null);
      const el = internalRef.current;
      if (event == null || el == null) return [];
      const { location, index } = resolveRegion(el, event);
      if (Haul.filterByType(Haul.FILE_TYPE, items).length > 0) {
        onFileDrop?.({ nodeKey, location, event });
        return items;
      }
      const dropped = filterTabDropHaulItems(items);
      if (dropped.length > 0)
        onDrop?.({ nodeKey, tabKey: dropped[0].key, location, index });
      const created = filterTabCreateHaulItems(items);
      if (created.length > 0)
        onCreate?.({
          nodeKey,
          location,
          tabKeys: created.map(({ key }) => key),
          index,
        });
      return dropped;
    },
    [nodeKey, onDrop, onCreate, onFileDrop],
  );

  const handleDragOver = useCallback(({ event }: Haul.OnDragOverProps): void => {
    const el = internalRef.current;
    if (event == null || el == null) return;
    const { location, index } = resolveRegion(el, event);
    if (index != null) {
      setMask(null);
      setInsertionIndex(index);
    } else {
      setMask(location);
      setInsertionIndex(null);
    }
  }, []);

  const haulProps = Haul.useDrop({
    type: "Mosaic",
    key: nodeKey,
    canDrop,
    onDrop: handleDrop,
    onDragOver: handleDragOver,
  });

  const handleDragLeave: DragEventHandler<HTMLDivElement> = useCallback(
    (e) => {
      onDragLeave?.(e);
      setMask(null);
      setInsertionIndex(null);
    },
    [onDragLeave],
  );

  return (
    <Flex.Box
      ref={combinedRef}
      className={CSS(CSS.BE("mosaic", "leaf"), className)}
      onDragLeave={handleDragLeave}
      empty
      {...haulProps}
      {...rest}
    >
      <Tabs.InsertionIndexProvider value={insertionIndex}>
        {children}
      </Tabs.InsertionIndexProvider>
      {mask != null && (
        <div className={CSS.BE("mosaic", "mask")} style={MASK_STYLE[mask]} />
      )}
    </Flex.Box>
  );
};
