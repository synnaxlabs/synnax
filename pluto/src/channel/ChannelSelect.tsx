// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useMemo } from "react";

import { ChannelKey, ChannelPayload } from "@synnaxlabs/client";

import { Client } from "..";

import { ListColumn, Select, SelectMultipleProps, SelectProps, Status } from "@/core";

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

  return (
    <Select.Multiple
      searcher={client?.channels}
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

export const ChannelSelect = ({
  columns: filter = [],
  ...props
}: ChannelSelectProps): ReactElement => {
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

  return (
    <Select
      searcher={client?.channels}
      columns={columns}
      emptyContent={emptyContent}
      tagKey={"name"}
      {...props}
    />
  );
};
