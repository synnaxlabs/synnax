// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { location } from "@synnaxlabs/x";
import {
  type ControlLinePosition,
  type ControlPosition,
  NodeResizeControl,
  ResizeControlVariant,
} from "@xyflow/react";
import {
  cloneElement,
  type CSSProperties,
  type DragEvent,
  type FC,
  Fragment,
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useMemo,
  useRef,
  useState,
} from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Haul } from "@/haul";
import { useSyncedRef } from "@/hooks";
import { Icon } from "@/icon";
import { triggerReflow } from "@/util/reflow";
import { selectNode } from "@/vis/diagram/util";

export interface GridItem {
  key: string;
  element: ReactElement<{
    style?: CSSProperties;
    draggable?: boolean;
    onDragStart?: (e: DragEvent<HTMLElement>) => void;
    onDragEnd?: (e: DragEvent<HTMLElement>) => void;
  }>;
  location: location.Location;
}

export interface GridProps extends PropsWithChildren<{}> {
  editable: boolean;
  symbolKey: string;
  items: GridItem[];
  onLocationChange: (key: string, loc: location.Location) => void;
  onRotate?: () => void;
  allowCenter?: boolean;
  allowRotate?: boolean;
  onResize?: (dimensions: { width: number; height: number }) => void;
}

interface GridElProps {
  editable: boolean;
  symbolKey: string;
  items: GridItem[];
  onLocationChange: (key: string, loc: location.Location) => void;
}

export const roundResizeDims = (
  width: number,
  height: number,
): { width: number; height: number } => ({
  width: Math.round(width),
  height: Math.round(height),
});

const RESIZE_CONTROLS: {
  position: ControlLinePosition | ControlPosition;
  variant?: ResizeControlVariant;
  keepAspectRatio?: boolean;
}[] = [
  { position: "top", variant: ResizeControlVariant.Line },
  { position: "right", variant: ResizeControlVariant.Line },
  { position: "bottom", variant: ResizeControlVariant.Line },
  { position: "left", variant: ResizeControlVariant.Line },
  { position: "top-left", keepAspectRatio: true },
  { position: "top-right", keepAspectRatio: true },
  { position: "bottom-left", keepAspectRatio: true },
  { position: "bottom-right", keepAspectRatio: true },
];

const HAUL_TYPE = "Schematic.Grid";

export const DRAG_HANDLE_CLASS = CSS.B("drag-handle");

const reflowPane = (symbolKey: string) => {
  const node = selectNode(symbolKey);
  const nearestDiagram = node.closest(".react-flow__pane");
  if (nearestDiagram != null) triggerReflow(nearestDiagram as HTMLElement);
};

const createGridEl = (loc: location.Location): FC<GridElProps> => {
  const EditableGridEl = ({
    symbolKey,
    items: fItems,
    onLocationChange,
  }: GridElProps): ReactElement | null => {
    const haulType = `${symbolKey}.${HAUL_TYPE}`;
    const [draggingOver, setDraggingOver] = useState(false);
    const canDrop: Haul.CanDrop = useMemo(
      () => Haul.canDropOfType(haulType),
      [haulType],
    );
    const onLocationChangeRef = useSyncedRef(onLocationChange);
    const { startDrag, onDragEnd, ...dropProps } = Haul.useDragAndDrop({
      type: haulType,
      canDrop,
      onDrop: useCallback(({ items }) => {
        setDraggingOver(false);
        return items;
      }, []),
      onDragOver: useCallback((props: Haul.OnDragOverProps) => {
        const { items } = props;
        setDraggingOver(canDrop(props));
        items.forEach(({ key }) => onLocationChangeRef.current(key as string, loc));
      }, []),
    });

    const items = fItems.filter((i) => i.location === loc);

    const onDragStart = useCallback(
      (_: DragEvent<HTMLElement>, key: string) => {
        startDrag([{ key, type: haulType }]);
        // We need to mount this listener because the onDragEnd will not fire if the
        // element is dragged to a different grid element and then released.
        document.addEventListener("mousemove", onDragEnd, { once: true });
      },
      [startDrag, haulType, onDragEnd],
    );

    const isDragging = canDrop(Haul.useDraggingState());

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
        {items.map(({ element, key }) => (
          <Fragment key={key}>
            {cloneElement(element, {
              draggable: true,
              onDragStart: (e: DragEvent<HTMLElement>) => onDragStart(e, key),
              onDragEnd,
              style: { ...element.props.style, cursor: "grab" },
            })}
          </Fragment>
        ))}
      </Flex.Box>
    );
  };

  const GridEl = ({ symbolKey, ...rest }: GridElProps): ReactElement | null => {
    const { editable, items: fItems } = rest;

    if (editable) return <EditableGridEl symbolKey={symbolKey} {...rest} />;
    const items = fItems.filter((i) => i.location === loc);
    if (items.length === 0) return null;
    return (
      <Flex.Box
        direction={location.direction(loc)}
        className={CSS(CSS.BE("grid", "item"), CSS.loc(loc))}
        empty
      >
        {items.map(({ element, key }) => (
          <Fragment key={key}>{element}</Fragment>
        ))}
      </Flex.Box>
    );
  };
  return GridEl;
};

const TopGridEl = createGridEl("top");
const LeftGridEl = createGridEl("left");
const RightGridEl = createGridEl("right");
const BottomGridEl = createGridEl("bottom");
const CenterGridEl = createGridEl("center");

export const Grid = ({
  editable,
  allowRotate = true,
  children,
  allowCenter,
  onRotate,
  symbolKey,
  onResize,
  ...rest
}: GridProps) => {
  const prevEditable = useRef(editable);
  if (editable !== prevEditable.current) {
    reflowPane(symbolKey);
    prevEditable.current = editable;
  }

  return (
    <>
      <TopGridEl
        key={`top-${symbolKey}`}
        editable={editable}
        symbolKey={symbolKey}
        {...rest}
      />
      <LeftGridEl
        key={`left-${symbolKey}`}
        editable={editable}
        symbolKey={symbolKey}
        {...rest}
      />
      <RightGridEl
        key={`right-${symbolKey}`}
        editable={editable}
        symbolKey={symbolKey}
        {...rest}
      />
      <BottomGridEl
        key={`bottom-${symbolKey}`}
        editable={editable}
        symbolKey={symbolKey}
        {...rest}
      />
      {allowCenter && (
        <CenterGridEl
          key={`center-${symbolKey}`}
          editable={editable}
          symbolKey={symbolKey}
          {...rest}
        />
      )}
      {editable && allowRotate && (
        <Button.Button
          className={CSS.BE("grid", "rotate")}
          size="tiny"
          variant="filled"
          onClick={onRotate}
        >
          <Icon.Rotate />
        </Button.Button>
      )}
      {onResize != null &&
        editable &&
        RESIZE_CONTROLS.map(({ position, ...rest }) => (
          <NodeResizeControl
            key={position}
            position={position}
            onResize={(_event, { width, height }) =>
              onResize?.(roundResizeDims(width, height))
            }
            {...rest}
          />
        ))}
      <div className={DRAG_HANDLE_CLASS}>{children}</div>
    </>
  );
};
