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
import { type z } from "zod";

import { useAddListener } from "@/sync/Context";

export const useListener = (
  channel: channel.Name,
  onChange: (series: MultiSeries) => void,
): void => {
  const addListener = useAddListener();
  useEffect(
    () =>
      addListener({
        channels: channel,
        handler: (frame) => onChange(frame.get(channel)),
      }),
    [addListener, channel, onChange],
  );
};

export const useParsedListener = <Z extends z.ZodTypeAny>(
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
