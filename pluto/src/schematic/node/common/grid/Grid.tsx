// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/common/grid/grid.css";

import { Button } from "@synnaxlabs/charon/button";
import { CSS } from "@synnaxlabs/charon/css";
import { Flex } from "@synnaxlabs/charon/flex";
import { Haul } from "@synnaxlabs/charon/haul";
import { useSyncedRef } from "@synnaxlabs/charon/hooks";
import { Icon } from "@synnaxlabs/charon/icon";
import { triggerReflow } from "@synnaxlabs/charon/util";
import { location } from "@synnaxlabs/x";
import {
  Children,
  cloneElement,
  type CSSProperties,
  type DragEvent,
  type FC,
  Fragment,
  isValidElement,
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
  useCallback,
  useMemo,
  useRef,
  useState,
} from "react";

import { selectNode } from "@/vis/diagram/util";

type DraggableElement = ReactElement<{
  style?: CSSProperties;
  draggable?: boolean;
  onDragStart?: (e: DragEvent<HTMLElement>) => void;
  onDragEnd?: (e: DragEvent<HTMLElement>) => void;
}>;

export interface ItemProps {
  itemKey: string;
  location: location.Location;
  onLocationChange?: (loc: location.Location) => void;
  children: DraggableElement;
}

const TAG = Symbol.for("pluto.grid.item");

const tag = <T,>(component: T): T => {
  (component as unknown as Record<symbol, true>)[TAG] = true;
  return component;
};

export const Item: FC<ItemProps> = tag(({ children }) => children);

export const createItem = <P extends {}>(component: FC<P>): FC<P> => tag(component);

const resolveItem = (child: ReactNode): ReactElement<ItemProps> | null => {
  if (!isValidElement(child)) return null;
  if (!(child.type as unknown as Record<symbol, true>)?.[TAG]) return null;
  if (child.type === Item) return child as ReactElement<ItemProps>;
  const rendered = (child.type as (props: unknown) => ReactNode)(child.props);
  return isValidElement(rendered) && rendered.type === Item
    ? (rendered as ReactElement<ItemProps>)
    : null;
};

const splitChildren = (
  children: ReactNode,
): { items: ItemProps[]; body: ReactNode[] } => {
  const items: ItemProps[] = [];
  const body: ReactNode[] = [];
  Children.forEach(children, (child) => {
    const item = resolveItem(child);
    if (item != null) items.push(item.props);
    else body.push(child);
  });
  return { items, body };
};

export interface GridProps extends PropsWithChildren<{}> {
  editable: boolean;
  nodeKey: string;
  orientation?: location.Outer;
  onRotate?: (params: { orientation: location.Outer }) => void;
  allowCenter?: boolean;
  allowRotate?: boolean;
}

const HAUL_TYPE = "schematic_grid";
export const DRAG_HANDLE_CLASS = CSS.B("drag-handle");

const reflowPane = (nodeKey: string) => {
  const node = selectNode(nodeKey);
  const nearestDiagram = node.closest(".react-flow__pane");
  if (nearestDiagram != null) triggerReflow(nearestDiagram as HTMLElement);
};

interface SlotProps {
  loc: location.Location;
  editable: boolean;
  nodeKey: string;
  items: ItemProps[];
}

const EditableSlot = ({ loc, nodeKey, items }: SlotProps) => {
  const haulType = `${nodeKey}_${HAUL_TYPE}`;
  const [draggingOver, setDraggingOver] = useState(false);
  const canDrop = useMemo(() => Haul.canDropOfType(haulType), [haulType]);
  const itemsRef = useSyncedRef(items);
  const { startDrag, onDragEnd, ...dropProps } = Haul.useDragAndDrop({
    type: haulType,
    canDrop,
    onDrop: useCallback(({ items }) => {
      setDraggingOver(false);
      return items;
    }, []),
    onDragOver: useCallback(
      (props: Haul.OnDragOverProps) => {
        setDraggingOver(canDrop(props));
        props.items.forEach(({ key }) => {
          const item = itemsRef.current.find((i) => i.itemKey === key);
          item?.onLocationChange?.(loc);
        });
      },
      [canDrop, loc],
    ),
  });
  const onDragStart = useCallback(
    (_: DragEvent<HTMLElement>, itemKey: string) => {
      startDrag([{ key: itemKey, type: haulType }]);
      // The onDragEnd handler will not fire if the element is dragged into a
      // different grid slot before being released; this listener cleans up.
      document.addEventListener("mousemove", onDragEnd, { once: true });
    },
    [startDrag, haulType, onDragEnd],
  );
  const isDragging = canDrop(Haul.useDraggingState());
  const filtered = items.filter((i) => i.location === loc);
  return (
    <Flex.Box
      direction={location.direction(loc)}
      className={CSS(
        CSS.BE("grid", "item"),
        CSS.loc(loc),
        CSS.dropRegion(isDragging),
        draggingOver && CSS.B("dragging-over"),
        isDragging && CSS.B("dragging"),
      )}
      onDragLeave={() => setDraggingOver(false)}
      empty
      {...dropProps}
    >
      {filtered.map(({ children, itemKey }) => (
        <Fragment key={itemKey}>
          {cloneElement(children, {
            draggable: true,
            onDragStart: (e: DragEvent<HTMLElement>) => onDragStart(e, itemKey),
            onDragEnd,
            style: { ...children.props.style, cursor: "grab" },
          })}
        </Fragment>
      ))}
    </Flex.Box>
  );
};

const StaticSlot = ({ loc, items }: SlotProps) => {
  const filtered = items.filter((i) => i.location === loc);
  if (filtered.length === 0) return null;
  return (
    <Flex.Box
      direction={location.direction(loc)}
      className={CSS(CSS.BE("grid", "item"), CSS.loc(loc))}
      empty
    >
      {filtered.map(({ children, itemKey }) => (
        <Fragment key={itemKey}>{children}</Fragment>
      ))}
    </Flex.Box>
  );
};

const Zone = (props: SlotProps) =>
  props.editable ? <EditableSlot {...props} /> : <StaticSlot {...props} />;

export const Grid: FC<GridProps> = ({
  editable,
  allowRotate = true,
  children,
  allowCenter = false,
  onRotate,
  nodeKey,
  orientation = "left",
}) => {
  const prevEditable = useRef(editable);
  if (editable !== prevEditable.current) {
    reflowPane(nodeKey);
    prevEditable.current = editable;
  }
  const { items, body } = splitChildren(children);
  const handleRotate = () =>
    onRotate?.({ orientation: location.rotate(orientation, "clockwise") });
  return (
    <>
      <Zone key="top" loc="top" editable={editable} nodeKey={nodeKey} items={items} />
      <Zone
        key="bottom"
        loc="bottom"
        editable={editable}
        nodeKey={nodeKey}
        items={items}
      />
      <Zone key="left" loc="left" editable={editable} nodeKey={nodeKey} items={items} />
      <Zone
        key="right"
        loc="right"
        editable={editable}
        nodeKey={nodeKey}
        items={items}
      />
      {allowCenter && (
        <Zone loc="center" editable={editable} nodeKey={nodeKey} items={items} />
      )}
      {editable && allowRotate && (
        <Button.Button
          className={CSS.BE("grid", "rotate")}
          size="tiny"
          variant="filled"
          onClick={handleRotate}
        >
          <Icon.Rotate />
        </Button.Button>
      )}
      <div className={DRAG_HANDLE_CLASS}>{body}</div>
    </>
  );
};
