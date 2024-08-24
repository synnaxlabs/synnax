// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { AsyncTermSearcher, nullToArr, toArray, unique } from "@synnaxlabs/x";
import {
  type DragEvent,
  FC,
  type ReactElement,
  useCallback,
  useId,
  useMemo,
} from "react";

import { CSS } from "@/css";
import { Haul } from "@/haul";
import { type DraggingState } from "@/haul/Haul";
import { HAUL_TYPE } from "@/label/types";
import { type List } from "@/list";
import { Select } from "@/select";
import { Status } from "@/status";
import { Synnax } from "@/synnax";
import { Tag } from "@/tag";
import { componentRenderProp } from "@/util/renderProp";

const rangeCols: Array<List.ColumnSpec<label.Key, label.Label>> = [
  {
    key: "color",
    name: "Color",
    render: ({ entry }) => (
      <Icon.Circle
        name="circle"
        color={entry.color}
        style={{ height: "2.5rem", width: "2.5rem" }}
      />
    ),
  },
  {
    key: "name",
    name: "Name",
  },
];

const canDrop = (
  { items: entities }: DraggingState,
  value: label.Key[] | readonly label.Key[],
): boolean => {
  const f = Haul.filterByType(HAUL_TYPE, entities);
  return f.length > 0 && !f.every((h) => value.includes(h.key as label.Key));
};

export interface SelectMultipleProps
  extends Omit<Select.MultipleProps<label.Key, label.Label>, "columns" | "searcher"> {}

const RenderTag: FC<Select.MultipleTagProps<label.Key, label.Label>> = ({
  entry,
  loading,
  entryKey: _,
  entryRenderKey: __,
  ...props
}) => (
  <Tag.Tag color={entry?.color} {...props}>
    {entry?.name}
  </Tag.Tag>
);

const renderTag = componentRenderProp(RenderTag);

export const SelectMultiple = ({
  onChange,
  className,
  value,
  ...props
}: SelectMultipleProps): ReactElement => {
  const client = Synnax.use();
  const emptyContent =
    client != null ? undefined : (
      <Status.Text.Centered variant="error" level="h4" style={{ height: 150 }}>
        No client available
      </Status.Text.Centered>
    );

  const {
    startDrag,
    onDragEnd: endDrag,
    ...dropProps
  } = Haul.useDragAndDrop({
    type: "Label.SelectMultiple",
    canDrop: useCallback((hauled) => canDrop(hauled, toArray(value)), [value]),
    onDrop: useCallback(
      ({ items }) => {
        const dropped = Haul.filterByType(HAUL_TYPE, items);
        if (dropped.length === 0) return [];
        const v = unique([
          ...toArray(value),
          ...(dropped.map((c) => c.key) as label.Key[]),
        ]);
        onChange(v, {
          clickedIndex: null,
          clicked: null,
          entries: [],
        });
        return dropped;
      },
      [onChange, value],
    ),
  });
  const dragging = Haul.useDraggingState();

  const handleSuccessfulDrop = useCallback(
    ({ dropped }: Haul.OnSuccessfulDropProps) => {
      onChange(
        toArray(value).filter((key) => !dropped.some((h) => h.key === key)),
        {
          clickedIndex: null,
          clicked: null,
          entries: [],
        },
      );
    },
    [onChange, value],
  );

  const onDragStart = useCallback(
    (_: DragEvent<HTMLDivElement>, key: label.Key) =>
      startDrag([{ key, type: HAUL_TYPE }], handleSuccessfulDrop),
    [startDrag, handleSuccessfulDrop],
  );

  return (
    <Select.Multiple
      className={CSS(className, CSS.dropRegion(canDrop(dragging, toArray(value))))}
      value={value}
      onTagDragStart={onDragStart}
      onTagDragEnd={endDrag}
      searcher={client?.labels}
      onChange={onChange}
      columns={rangeCols}
      emptyContent={emptyContent}
      renderTag={renderTag}
      addPlaceholder="Label"
      {...dropProps}
      {...props}
    />
  );
};

export interface SelectSingleProps
  extends Omit<Select.SingleProps<label.Key, label.Label>, "columns"> {}

interface UseSingleReturn extends Haul.UseDragReturn {
  emptyContent?: ReactElement;
  dragging: DraggingState;
  onDragStart: (e: DragEvent<HTMLDivElement>) => void;
  searcher?: AsyncTermSearcher<string, label.Key, label.Label>;
}

const useSingle = ({
  value,
  onChange,
}: Pick<SelectSingleProps, "onChange" | "value">): UseSingleReturn => {
  const client = Synnax.use();
  const emptyContent =
    client != null ? undefined : (
      <Status.Text.Centered variant="error" level="h4" style={{ height: 150 }}>
        No client available
      </Status.Text.Centered>
    );

  const id = useId();
  const sourceAndTarget: Haul.Item = useMemo(
    () => ({ key: id, type: "Label.SelectMultiple" }),
    [id],
  );

  const dragProps = Haul.useDragAndDrop({
    type: "Label.SelectSingle",
    canDrop: useCallback((hauled) => canDrop(hauled, nullToArr(value)), [value]),
    onDrop: useCallback(
      ({ items }) => {
        const ch = Haul.filterByType(HAUL_TYPE, items);
        if (ch.length === 0) return [];
        onChange(ch[0].key as label.Key, {
          clickedIndex: null,
          clicked: null,
          entries: [],
        });
        return ch;
      },
      [sourceAndTarget, onChange],
    ),
  });

  const dragging = Haul.useDraggingState();
  const onDragStart = useCallback(
    () => value != null && dragProps.startDrag([{ type: HAUL_TYPE, key: value }]),
    [dragProps.startDrag, value],
  );
  return {
    emptyContent,
    dragging,
    ...dragProps,
    onDragStart,
    searcher: client?.labels,
  };
};

export const SelectSingle = ({
  onChange,
  value,
  className,
  data,
  ...props
}: SelectSingleProps): ReactElement => {
  const { dragging, ...dragProps } = useSingle({ value, onChange });
  return (
    <Select.Single<label.Key, label.Label>
      data={data}
      className={CSS(className, CSS.dropRegion(canDrop(dragging, nullToArr(value))))}
      value={value}
      onChange={onChange}
      columns={rangeCols}
      entryRenderKey={"name"}
      {...dragProps}
      {...props}
    />
  );
};

export const SelectButton = ({
  data,
  value,
  onChange,
  ...props
}: SelectSingleProps): ReactElement => {
  const { dragging, ...dragProps } = useSingle({ value, onChange });
  return (
    <Select.Single<label.Key, label.Label>
      data={data}
      value={value as string}
      onChange={onChange}
      columns={rangeCols}
      entryRenderKey={"name"}
      {...dragProps}
      {...props}
    />
  );
};
