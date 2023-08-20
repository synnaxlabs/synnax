// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DragEvent, ReactElement, useCallback, useMemo } from "react";

import { ChannelKey, ChannelPayload } from "@synnaxlabs/client";
import { unique } from "@synnaxlabs/x";

import { Client } from "@/client/main";
import {
  CSS,
  ListColumn,
  Select,
  SelectMultipleProps,
  SelectProps,
  Status,
} from "@/core";
import { Haul } from "@/haul";

const channelColumns: Array<ListColumn<ChannelKey, ChannelPayload>> = [
  {
    key: "name",
    name: "Name",
  },
  {
    key: "rate",
    name: "Rate",
  },
  {
    key: "dataType",
    name: "Data Type",
  },
  {
    key: "index",
    name: "Index",
  },
  {
    key: "key",
    name: "Key",
  },
  {
    key: "isIndex",
    name: "Is Index",
  },
];

export interface ChannelSelectMultipleProps
  extends Omit<
    SelectMultipleProps<ChannelKey, ChannelPayload>,
    "columns" | "searcher"
  > {
  columns?: string[];
}

export const ChannelSelectMultiple = ({
  columns: filter = [],
  onChange,
  value,
  className,
  ...props
}: ChannelSelectMultipleProps): ReactElement => {
  const client = Client.use();
  const columns = useMemo(() => {
    if (filter.length === 0) return channelColumns;
    return channelColumns.filter((column) => filter.includes(column.key));
  }, [filter]);

  const emptyContent =
    client != null ? undefined : (
      <Status.Text.Centered variant="error" level="h4">
        No client available
      </Status.Text.Centered>
    );

  const { onDragOver, onDrop } = Haul.useDrop({
    canDrop: useCallback((hauled) => canDrop(hauled, value), [value]),
    onDrop: useCallback(
      ([channel]) => onChange(unique([...value, channel.key as ChannelKey])),
      [onChange, value]
    ),
  });
  const { startDrag, endDrag } = Haul.useDrag();
  const dragging = Haul.useDraggingState();

  const handleSuccessfulDrop = useCallback(
    (dragging: Haul.Item[]) => {
      onChange(value.filter((key) => !dragging.some((h) => h.key === key)));
    },
    [onChange, value]
  );

  const onDragStart = useCallback(
    (_: DragEvent<HTMLDivElement>, key: ChannelKey) =>
      startDrag([{ key, type: "channel" }], handleSuccessfulDrop),
    [startDrag, handleSuccessfulDrop]
  );

  return (
    <Select.Multiple
      className={CSS(className, CSS.dropRegion(canDrop(dragging, value)))}
      value={value}
      onDragOver={onDragOver}
      onDrop={onDrop}
      onTagDragStart={onDragStart}
      onTagDragEnd={endDrag}
      searcher={client?.channels}
      onChange={onChange}
      columns={columns}
      emptyContent={emptyContent}
      tagKey={"name"}
      {...props}
    />
  );
};

export interface ChannelSelectProps
  extends Omit<SelectProps<ChannelKey, ChannelPayload>, "columns"> {
  columns?: string[];
}

const canDrop = (
  hauled: Haul.Item[],
  value: ChannelKey[] | readonly ChannelKey[]
): boolean =>
  hauled.length > 0 &&
  hauled.every((h) => h.type === "channel") &&
  !hauled.every((h) => value.includes(h.key as ChannelKey));

export const ChannelSelect = ({
  columns: filter = [],
  onChange,
  value,
  className,
  ...props
}: ChannelSelectProps): ReactElement => {
  const client = Client.use();
  const columns = useMemo(() => {
    if (filter.length === 0) return channelColumns;
    return channelColumns.filter((column) => filter.includes(column.key));
  }, [filter]);

  const emptyContent =
    client != null ? undefined : (
      <Status.Text.Centered variant="error" level="h4" style={{ height: 150 }}>
        No client available
      </Status.Text.Centered>
    );

  const { onDragOver, onDrop } = Haul.useDrop({
    canDrop: useCallback((hauled) => canDrop(hauled, [value]), [value]),
    onDrop: useCallback(([channel]) => onChange(channel.key as ChannelKey), [onChange]),
  });

  const { startDrag, endDrag } = Haul.useDrag();
  const dragging = Haul.useDraggingState();
  const onDragStart = useCallback(() => {
    startDrag([{ type: "channel", key: value }]);
  }, [startDrag, value]);

  return (
    <Select
      className={CSS(className, CSS.dropRegion(canDrop(dragging, [value])))}
      value={value}
      onDragStart={onDragStart}
      onDragEnd={endDrag}
      onDragOver={onDragOver}
      onDrop={onDrop}
      onChange={onChange}
      searcher={client?.channels}
      columns={columns}
      emptyContent={emptyContent}
      tagKey={"name"}
      {...props}
    />
  );
};
