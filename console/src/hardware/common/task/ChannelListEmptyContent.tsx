// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Align, Text } from "@synnaxlabs/pluto";

export interface ChannelListEmptyContentProps {
  onAdd: () => void;
  snapshot?: boolean;
}

export const ChannelListEmptyContent = ({
  onAdd,
  snapshot = false,
}: ChannelListEmptyContentProps) => (
  <Align.Center direction="y" justify="center">
    <Text.Text level="p">No channels in task.</Text.Text>
    {!snapshot && (
      <Text.Link level="p" onClick={onAdd}>
        Add a channel
      </Text.Link>
    )}
  </Align.Center>
);
