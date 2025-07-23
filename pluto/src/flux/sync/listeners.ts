// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType, type MultiSeries } from "@synnaxlabs/client";
import { array } from "@synnaxlabs/x";
import { Mutex } from "async-mutex";
import { useEffect } from "react";
import { type z } from "zod";

import { useAddListener } from "@/flux/sync/Context";
import { Status } from "@/status";

export type ListenerArgs<Value, Extra> = {
  changed: Value;
} & Extra;

export interface ListenerHandler<Value, Extra> {
  (args: ListenerArgs<Value, Extra>): Promise<unknown> | unknown;
}

export const parsedHandler =
  <Z extends z.ZodType, Extra>(
    schema: Z,
    onChange: ListenerHandler<z.output<Z>, Extra>,
  ): ListenerHandler<MultiSeries, Extra> =>
  async (args) => {
    let parsed: z.output<Z>[];
    if (!args.changed.dataType.equals(DataType.JSON))
      parsed = args.changed.toStrings().map((s) => schema.parse(s));
    else parsed = args.changed.parseJSON(schema);
    for (const value of parsed) await onChange({ ...args, changed: value });
  };

export const stringHandler =
  <Extra>(
    onChange: ListenerHandler<string, Extra>,
  ): ListenerHandler<MultiSeries, Extra> =>
  async (args) => {
    for (const value of args.changed.toStrings())
      await onChange({ ...args, changed: value });
  };

export interface ListenerSpec<Value, Extra> {
  channel: channel.Name;
  onChange: ListenerHandler<Value, Extra>;
}

export const useListener = (
  listeners: ListenerSpec<MultiSeries, {}> | ListenerSpec<MultiSeries, {}>[],
): void => {
  const addListener = useAddListener();
  const handleError = Status.useErrorHandler();
  useEffect(() => {
    const mu = new Mutex();
    const destructors = array.toArray(listeners).map(({ channel, onChange }) =>
      addListener({
        channel,
        handler: (frame) => {
          handleError(async () => {
            await mu.runExclusive(async () => {
              await onChange({ changed: frame.get(channel) });
            });
          }, "Error in Sync.useListener");
        },
      }),
    );
    return () => {
      for (const destructor of destructors) destructor();
    };
  }, [addListener]);
};
