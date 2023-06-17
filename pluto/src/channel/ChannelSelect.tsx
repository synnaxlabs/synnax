// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { ChannelKey, ChannelPayload } from "@synnaxlabs/client";

import { ListColumn, SelectMultipleProps } from "@/core";

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
    key: "rate",
    name: "Rate",
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

export interface ChannelSelectMultipleProps extends Omit<SelectMultipleProps<ChannelKey, ChannelPayload>, "columns"> {
    columns: string[];
}

export const ChannelsSelectMultiple = (props: ChannelSelectMultipleProps): ReactElement => (
