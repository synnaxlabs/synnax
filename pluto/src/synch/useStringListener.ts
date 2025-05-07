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

import { useListener } from "@/synch/useListener";

export const useStringListener = <T>(
  channel: channel.Name,
  parseString: (str: string) => T,
  onUpdate: (value: T) => void,
) => {
  const handleUpdate = useCallback(
    ({ series }: MultiSeries) =>
      series
        .flatMap((s) => s.toStrings())
        .map(parseString)
        .forEach(onUpdate),
    [parseString, onUpdate],
  );
  useListener(channel, handleUpdate);
};
