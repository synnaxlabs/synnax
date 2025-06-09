// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { compare, location } from "@synnaxlabs/x";
import {
  cloneElement,
  type CSSProperties,
  type DragEvent,
  type FC,
  Fragment,
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useRef,
  useState,
} from "react";

import { Align } from "@/align";
import { Button } from "@/button";
import { CSS } from "@/css";
import { Haul } from "@/haul";
import { useSyncedRef } from "@/hooks";
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
  includeCenter?: boolean;
}

interface GridElProps {
  editable: boolean;
  symbolKey: string;
  items: GridItem[];
  onLocationChange: (key: string, loc: location.Location) => void;
}

const HAUL_TYPE = "Schematic.Grid";

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
    const canDrop: Haul.CanDrop = Haul.canDropOfType(haulType);
    const onLocationChangeRef = useSyncedRef(onLocationChange);
    const { startDrag, onDragEnd, ...dropProps } = Haul.useDragAndDrop({
      type: haulType,
      canDrop,
      onDrop: useCallback(({ items }) => {
        setDraggingOver(false);
        return items;
      }, []),
      onDragOver: useCallback((props: Haul.OnDragOverProps) => {
        setDraggingOver(canDrop(props));
        props.items.forEach(({ key }) =>
          onLocationChangeRef.current(key as string, loc),
        );
      }, []),
    });

    const items = fItems.filter((i) => i.location === loc);

    const onDragStart = useCallback(
      (e: DragEvent<HTMLElement>, key: string) => {
        e.stopPropagation();
        startDrag([{ key, type: haulType }]);
      },
      [startDrag, haulType],
    );

    const isDragging = canDrop(Haul.useDraggingState());

    return (
      <Align.Space
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
      </Align.Space>
    );
  };

  const GridEl = ({ symbolKey, ...rest }: GridElProps): ReactElement | null => {
    const { editable, items: fItems } = rest;

    const itemKeys = fItems.map((i) => i.key);
    const prevItemKeys = useRef(itemKeys);
    if (compare.primitiveArrays(itemKeys, prevItemKeys.current)) {
      reflowPane(symbolKey);
      prevItemKeys.current = itemKeys;
    }

    if (editable) return <EditableGridEl symbolKey={symbolKey} {...rest} />;
    const items = fItems.filter((i) => i.location === loc);
    if (items.length === 0) return null;
    return (
      <Align.Space
        direction={location.direction(loc)}
        className={CSS(CSS.BE("grid", "item"), CSS.loc(loc))}
        empty
      >
        {items.map(({ element, key }) => (
          <Fragment key={key}>{element}</Fragment>
        ))}
      </Align.Space>
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
  onRotate,
  children,
  includeCenter = false,
  ...rest
}: GridProps) => (
  <>
    <TopGridEl editable={editable} {...rest} />
    <LeftGridEl editable={editable} {...rest} />
    <RightGridEl editable={editable} {...rest} />
    <BottomGridEl editable={editable} {...rest} />
    {includeCenter && <CenterGridEl editable={editable} {...rest} />}
    {editable && (
      <Button.Icon
        className={CSS.BE("grid", "rotate")}
        size="small"
        variant="filled"
        onClick={onRotate}
      >
        <Icon.Rotate />
      </Button.Icon>
    )}
    {children}
  </>
);
