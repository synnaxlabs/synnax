// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { location } from "@synnaxlabs/x";
import {
  cloneElement,
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
  element: ReactElement;
  location: location.Outer;
}

export interface GridProps extends PropsWithChildren<{}> {
  editable: boolean;
  symbolKey: string;
  items: GridItem[];
  onLocationChange: (key: string, loc: location.Outer) => void;
  onRotate?: () => void;
}

interface GridElProps {
  editable: boolean;
  symbolKey: string;
  items: GridItem[];
  onLocationChange: (key: string, loc: location.Outer) => void;
}

const HAUL_TYPE = "Schematic.Grid";

const reflowPane = (symbolKey: string) => {
  const node = selectNode(symbolKey);
  const nearestDiagram = node.closest(".react-flow__pane") as HTMLElement | null;
  if (nearestDiagram != null) triggerReflow(nearestDiagram);
};

const createGridEl = (loc: location.Outer): FC<GridElProps> => {
  const EditableGridEl = ({
    symbolKey,
    items: fItems,
    onLocationChange,
  }: GridElProps): ReactElement | null => {
    const haulType = `${symbolKey}.${HAUL_TYPE}`;
    const [draggingOver, setDraggingOver] = useState(false);
    const canDrop: Haul.CanDrop = useCallback(
      ({ items }) => items.every((i) => i.type === haulType),
      [haulType],
    );
    const onLocationChangeRef = useSyncedRef(onLocationChange);
    const { startDrag, onDragEnd, ...dropProps } = Haul.useDragAndDrop({
      type: haulType,
      canDrop,
      onDrop: useCallback(({ items }) => items, []),
      onDragOver: useCallback((props: Haul.OnDragOverProps) => {
        setDraggingOver(canDrop(props));
        props.items.forEach(({ key }) =>
          onLocationChangeRef.current(key as string, loc),
        );
      }, []),
    });

    const items = fItems.filter((i) => i.location === loc);

    const onDragStart = useCallback(
      (_: DragEvent<HTMLDivElement>, key: string) =>
        startDrag([{ key, type: haulType }]),
      [startDrag, haulType],
    );

    return (
      <Align.Space
        direction={location.direction(loc)}
        className={CSS(
          CSS.BE("grid", "item"),
          CSS.loc(loc),
          CSS.dropRegion(true),
          draggingOver && CSS.B("dragging-over"),
        )}
        onDragLeave={() => setDraggingOver(false)}
        empty
        {...dropProps}
      >
        {items.map(({ element, key }) => (
          <Fragment key={key}>
            {cloneElement(element, {
              draggable: true,
              onDragStart: (e: DragEvent<HTMLDivElement>) => onDragStart(e, key),
              onDragEnd,
              style: { ...element.props.style, cursor: "grab" },
            })}
          </Fragment>
        ))}
      </Align.Space>
    );
  };

  const GridEl = ({ symbolKey, ...props }: GridElProps): ReactElement | null => {
    const { editable, items: fItems } = props;

    const prevEditable = useRef(editable);
    if (editable !== prevEditable.current && loc === "top") {
      reflowPane(symbolKey);
      prevEditable.current = editable;
    }

    if (editable) return <EditableGridEl symbolKey={symbolKey} {...props} />;
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

export const Grid = ({ editable, onRotate, children, ...props }: GridProps) => (
  <>
    <TopGridEl editable={editable} {...props} />
    <LeftGridEl editable={editable} {...props} />
    <RightGridEl editable={editable} {...props} />
    <BottomGridEl editable={editable} {...props} />
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
