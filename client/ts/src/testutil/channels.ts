// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id } from "@synnaxlabs/x";

import type { channel } from "@/channel";
import type Synnax from "@/client";
import type { spyOnSend } from "@/testutil/client";

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

export const newIndexedBoolPair = async (
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
    dataType: DataType.BOOLEAN,
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

export const newVirtualBoolChannel = async (
  client: Synnax,
): Promise<channel.Channel> => {
  const ch = await client.channels.create({
    name: id.create(),
    dataType: DataType.BOOLEAN,
    virtual: true,
  });
  return ch;
};

/** @returns whether a request is a client's fetch of the built-in channel keys. */
export const isBuiltInsFetch = (target: string, req: unknown): boolean =>
  target === "/channel/retrieve" &&
  (req as channel.RetrieveRequest).where?.internal?.eq === true;

/**
 * @returns the response of each fetch of the built-in channel keys that send saw, in
 * the order the client sent them.
 */
export const builtInsFetches = (
  send: ReturnType<typeof spyOnSend>,
): Promise<unknown>[] =>
  send.mock.calls.flatMap(([target, req], i) =>
    isBuiltInsFetch(target, req)
      ? [send.mock.results[i].value as Promise<unknown>]
      : [],
  );
