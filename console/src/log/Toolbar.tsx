// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, log } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Channel, Input } from "@synnaxlabs/pluto";
import { ReactElement } from "react";
import { useDispatch } from "react-redux";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { useSelect } from "@/log/selectors";
import { setChannels } from "@/log/slice";

export interface ToolbarProps {
  layoutKey: string;
}

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement => {
  const d = useDispatch();
  const { name } = Layout.useSelectRequired(layoutKey);
  const state = useSelect(layoutKey);
  const handleChannelChange = (v: channel.Key) =>
    d(setChannels({ key: layoutKey, channels: [v ?? 0] }));
  return (
    <>
      <ToolbarHeader>
        <ToolbarTitle icon={<Icon.Log />}>{name}</ToolbarTitle>
        <Align.Space direction="x">
          <Link.ToolbarCopyButton
            name={name}
            ontologyID={{ key: state.key, type: log.ONTOLOGY_TYPE }}
          />
        </Align.Space>
      </ToolbarHeader>
      <Align.Space style={{ padding: "2rem", width: "100%" }} direction="x">
        <Input.Item label="Channel" grow>
          <Channel.SelectSingle
            value={state.channels[0]}
            onChange={handleChannelChange}
          />
        </Input.Item>
      </Align.Space>
    </>
  );
};
