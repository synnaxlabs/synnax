// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type MultiSeries } from "@synnaxlabs/client";
import { useCallback } from "react";
import { type z } from "zod";

import { useListener } from "@/synch/useListener";

export const useParsedListener = <Z extends z.ZodTypeAny>(
  channel: channel.Name,
  schema: Z,
  onUpdate: (value: z.infer<Z>) => void,
): void => {
  const handleUpdate = useCallback(
    (series: MultiSeries) => series.parseJSON(schema).forEach(onUpdate),
    [onUpdate, schema],
  );
  useListener(channel, handleUpdate);
};
