// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id } from "@synnaxlabs/x";

import { type channel } from "@/channel";
import type Synnax from "@/client";

export const newIndexedPair = async (
  client: Synnax,
): Promise<[channel.Channel, channel.Channel]> => {
  const index = await client.channels.create({
    leaseholder: 1,
    name: id.create(),
    dataType: DataType.TIMESTAMP,
    isIndex: true,
  });
  const data = await client.channels.create({
    leaseholder: 1,
    name: id.create(),
    dataType: DataType.FLOAT64,
    index: index.key,
  });
  return [index, data];
};

export const newVirtualChannel = async (client: Synnax): Promise<channel.Channel> => {
  const ch = await client.channels.create({
    name: id.create(),
    dataType: DataType.FLOAT64,
    virtual: true,
  });
  return ch;
};
