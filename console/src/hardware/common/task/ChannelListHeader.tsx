// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Header } from "@synnaxlabs/pluto";

import { type ChannelListEmptyContentProps } from "@/hardware/common/task/ChannelListEmptyContent";

export interface ChannelListHeaderProps extends ChannelListEmptyContentProps {}

export const ChannelListHeader = ({ onAdd, snapshot }: ChannelListHeaderProps) => (
  <Header.Header level="h4">
    <Header.Title weight={500}>Channels</Header.Title>
    {!snapshot && (
      <Header.Actions>
        {[{ key: "add", onClick: onAdd, children: <Icon.Add />, size: "large" }]}
      </Header.Actions>
    )}
  </Header.Header>
);
