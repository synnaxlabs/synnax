// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type MultiSeries } from "@synnaxlabs/client";
import { useEffect } from "react";

import { useAddListener } from "@/synch/useAddListener";

export const useListener = (
  channel: channel.Name,
  onUpdate: (series: MultiSeries) => void,
): void => {
  const addListener = useAddListener();
  useEffect(
    () =>
      addListener({
        channels: channel,
        handler: (frame) => onUpdate(frame.get(channel)),
      }),
    [addListener, channel, onUpdate],
  );
};
