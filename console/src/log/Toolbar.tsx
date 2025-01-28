// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, log } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Channel, Input } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { Export } from "@/export";
import { isDev } from "@/isDev";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { useExport } from "@/log/export";
import { useSyncComponent } from "@/log/Log";
import { useSelectOptional } from "@/log/selectors";
import { setChannels } from "@/log/slice";

export interface ToolbarProps {
  layoutKey: string;
}

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement | null => {
  const d = useSyncComponent(layoutKey);
  const { name } = Layout.useSelectRequired(layoutKey);
  const state = useSelectOptional(layoutKey);
  const handleChannelChange = (v: channel.Key) =>
    d(setChannels({ key: layoutKey, channels: [v ?? 0] }));
  const handleExport = useExport();
  if (state == null) return null;
  return (
    <>
      <ToolbarHeader>
        <ToolbarTitle icon={<Icon.Log />}>{name}</ToolbarTitle>
        <Align.Space direction="x" style={{ width: 66 }} empty>
          <Export.ToolbarButton onExport={() => handleExport(state.key)} />
          <Link.CopyToolbarButton name={name} ontologyID={log.ontologyID(state.key)} />
        </Align.Space>
      </ToolbarHeader>
      <Align.Space style={{ padding: "2rem", width: "100%" }} direction="x">
        <Input.Item label="Channel" grow>
          <Channel.SelectSingle
            value={state.channels[0]}
            onChange={handleChannelChange}
            searchOptions={{ internal: isDev() ? undefined : false }}
          />
        </Input.Item>
      </Align.Space>
    </>
  );
};
