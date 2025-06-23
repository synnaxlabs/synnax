// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type MultiSeries } from "@synnaxlabs/client";
import { useCallback, useEffect } from "react";
import { type z } from "zod/v4";

import { useSyncedRef } from "@/hooks";
import { useAddListener } from "@/sync/Context";
import { type ListenerHandler, type Params } from "@/sync/query";

export const parsedHandler =
  <P extends Params, Z extends z.ZodType, Value>(
    schema: Z,
    onChange: ListenerHandler<P, z.infer<Z>, Value>,
  ): ListenerHandler<P, MultiSeries, Value> =>
  (ctx) => {
    ctx.changed
      .parseJSON(schema)
      .forEach((value) => onChange({ ...ctx, changed: value }));
  };

export const useListener = (
  channel: channel.Name,
  onChange: (series: MultiSeries) => void,
): void => {
  const addListener = useAddListener();
  const onChangeRef = useSyncedRef(onChange);
  useEffect(
    () =>
      addListener({
        channels: channel,
        handler: (frame) => onChangeRef.current(frame.get(channel)),
      }),
    [addListener, channel],
  );
};

export const useParsedListener = <Z extends z.ZodType>(
  channel: channel.Name,
  schema: Z,
  onChange: (value: z.infer<Z>) => void,
): void => {
  const handleChange = useCallback(
    (series: MultiSeries) => series.parseJSON(schema).forEach(onChange),
    [onChange, schema],
  );
  useListener(channel, handleChange);
};

export const useStringListener = <T>(
  channel: channel.Name,
  parseString: (str: string) => T,
  onChange: (value: T) => void,
): void => {
  const handleChange = useCallback(
    ({ series }: MultiSeries) =>
      series
        .flatMap((s) => s.toStrings())
        .map(parseString)
        .forEach(onChange),
    [parseString, onChange],
  );
  useListener(channel, handleChange);
};
