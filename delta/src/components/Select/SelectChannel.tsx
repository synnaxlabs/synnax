// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import type { ChannelKey, ChannelPayload } from "@synnaxlabs/client";
import { Select } from "@synnaxlabs/pluto";
import type { ListColumn, SelectMultipleProps, SelectProps } from "@synnaxlabs/pluto";

const channelColumns: Array<ListColumn<ChannelKey, ChannelPayload>> = [
  {
    key: "name",
    name: "Name",
  },
];

const verboseChannelColumns: Array<ListColumn<ChannelKey, ChannelPayload>> = [
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
];

export interface SelectMultipleChannelsProps
  extends Omit<SelectMultipleProps<ChannelKey, ChannelPayload>, "columns"> {
  verbose?: boolean;
}

export const SelectMultipleChannels = ({
  verbose = false,
  ...props
}: SelectMultipleChannelsProps): ReactElement => (
  <Select.Multiple
    columns={verbose ? verboseChannelColumns : channelColumns}
    {...props}
  />
);

export interface SelectChannelProps
  extends Omit<SelectProps<ChannelKey, ChannelPayload>, "columns"> {
  verbose?: boolean;
}

export const SelectChanel = ({
  verbose = false,
  ...props
}: SelectChannelProps): ReactElement => (
  <Select columns={verbose ? verboseChannelColumns : channelColumns} {...props} />
);
