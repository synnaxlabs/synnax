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

const leafTabKeys = (el: HTMLElement): string[] =>
  Array.from(el.querySelectorAll(Tabs.KEY_SELECTOR))
    .map((tab) => tab.getAttribute(Tabs.KEY_ATTRIBUTE))
    .filter((key): key is string => key != null);

// The strip sits on the leaf's top edge, so measuring its cursor position against
// the leaf box would resolve a split; a drop on the strip means "into this leaf".
// Only file drags reach the leaf over the strip: the Selector claims tab drops.
const overTabStrip = (el: HTMLElement, cursor: xy.XY): boolean => {
  const strip = el.querySelector(Tabs.LIST_SELECTOR);
  return strip != null && box.contains(box.construct(strip), cursor);
};

const resolveLocation = (
  el: HTMLElement,
  event: React.DragEvent,
): location.Location => {
  if (leafTabKeys(el).length === 0) return "center";
  const cursor = xy.construct(event.clientX, event.clientY);
  if (overTabStrip(el, cursor)) return "center";
  const b = box.construct(el);
  return insertLocation({
    x: (cursor.x - box.left(b)) / box.width(b),
    y: (cursor.y - box.top(b)) / box.height(b),
  });
};

export interface LeafProps extends Omit<Flex.BoxProps, "onDrop" | "onDragOver"> {
  /** The key identifying this leaf, passed to the Frame's drop handlers. */
  nodeKey: number;
}

/**
 * Leaf is the haul drop target for the body of one pane of a mosaic. It resolves
 * the drop region from the cursor: an edge splits (showing a mask over the
 * affected half) and the middle drops in the center. Resolved drops are reported
 * to the Frame's onDrop, onCreate, and onFileDrop handlers with this leaf's key.
 * Drops on the pane's tab strip belong to the Tabs.Selector composed inside it:
 * wire it up with {@link useSelectorDropProps} so it claims strip drops (with an
 * insertion index) before they reach the leaf. OS file drags are the exception:
 * the strip rejects them, so they fall through to the leaf, which resolves the
 * strip region to a center drop.
 *
 * The leaf discovers its tabs from the DOM: tabs are elements with a
 * `data-tab-key` attribute, rendered by the Tabs parts composed inside it.
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
      const el = internalRef.current;
      if (event == null || el == null) return [];
      const location = resolveLocation(el, event);
      if (Haul.filterByType(Haul.FILE_TYPE, items).length > 0) {
        onFileDrop?.({ nodeKey, location, event });
        return items;
      }
      const dropped = filterTabDropHaulItems(items);
      if (dropped.length > 0) onDrop?.({ nodeKey, tabKey: dropped[0].key, location });
      const created = filterTabCreateHaulItems(items);
      if (created.length > 0)
        onCreate?.({ nodeKey, location, tabKeys: created.map(({ key }) => key) });
      return dropped;
    },
    [nodeKey, onDrop, onCreate, onFileDrop],
  );

  const handleDragOver = useCallback(({ event }: Haul.OnDragOverProps): void => {
    const el = internalRef.current;
    if (event == null || el == null) return;
    setMask(resolveLocation(el, event));
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
      {children}
      {mask != null && (
        <div
          className={CSS(CSS.BE("mosaic", "mask"), CSS.BEM("mosaic", "mask", mask))}
        />
      )}
    </Flex.Box>
  );
};
