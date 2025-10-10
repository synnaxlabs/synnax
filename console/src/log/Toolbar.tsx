// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, log } from "@synnaxlabs/client";
import { Channel, Flex, Icon, Input } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Cluster } from "@/cluster";
import { Toolbar as Core } from "@/components";
import { Export } from "@/export";
import { Layout } from "@/layout";
import { useExport } from "@/log/export";
import { useSyncComponent } from "@/log/Log";
import { useSelectOptional } from "@/log/selectors";
import { setChannels } from "@/log/slice";

export interface ToolbarProps {
  layoutKey: string;
}

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement | null => {
  const dispatch = useSyncComponent(layoutKey);
  const { name } = Layout.useSelectRequired(layoutKey);
  const state = useSelectOptional(layoutKey);
  const handleChannelChange = (v: channel.Key) =>
    dispatch(setChannels({ key: layoutKey, channels: [v ?? 0] }));
  const handleExport = useExport();
  if (state == null) return null;
  return (
    <Core.Content>
      <Core.Header>
        <Core.Title icon={<Icon.Log />}>{name}</Core.Title>
        <Flex.Box x style={{ width: 66 }} empty>
          <Export.ToolbarButton onExport={() => handleExport(state.key)} />
          <Cluster.CopyLinkToolbarButton
            name={name}
            ontologyID={log.ontologyID(state.key)}
          />
        </Flex.Box>
      </Core.Header>
      <Flex.Box full style={{ padding: "2rem" }}>
        <Input.Item label="Channel" grow>
          <Channel.SelectSingle
            value={state.channels[0]}
            onChange={handleChannelChange}
            initialQuery={{ internal: IS_DEV }}
          />
        </Input.Item>
      </Flex.Box>
    </Core.Content>
  );
};
