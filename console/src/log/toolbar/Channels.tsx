// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, log } from "@synnaxlabs/client";
import { Access, Channel, Flex, Input } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { useSyncComponent } from "@/log/Log";
import { useSelectOptional } from "@/log/selectors";
import { setChannels, setTimestampPrecision } from "@/log/slice";

export interface ChannelsProps {
  layoutKey: string;
}

export const Channels = ({ layoutKey }: ChannelsProps): ReactElement | null => {
  const dispatch = useSyncComponent(layoutKey);
  const state = useSelectOptional(layoutKey);
  const hasEditPermission = Access.useUpdateGranted(log.ontologyID(layoutKey));
  const handleChannelChange = (v: channel.Key[]) =>
    dispatch(setChannels({ key: layoutKey, channels: v }));
  const handlePrecisionChange = (v: number) =>
    dispatch(setTimestampPrecision({ key: layoutKey, timestampPrecision: v }));
  if (state == null) return null;
  return (
    <Flex.Box x style={{ padding: "2rem" }}>
      <Input.Item label="Channels" grow>
        <Channel.SelectMultiple
          value={state.channels}
          onChange={handleChannelChange}
          initialQuery={{ internal: IS_DEV ? undefined : false }}
          disabled={!hasEditPermission}
        />
      </Input.Item>
      <Input.Item label="Timestamp Precision">
        <Input.Numeric
          value={state.timestampPrecision}
          onChange={handlePrecisionChange}
          resetValue={0}
          bounds={{ lower: 0, upper: 4 }}
          disabled={!hasEditPermission}
        />
      </Input.Item>
    </Flex.Box>
  );
};
