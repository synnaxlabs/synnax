// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ethercat } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import {
  type Channel,
  entryAt,
  getPortLabel,
  hasEntry,
} from "@/feature/ethercat/task/types";

export interface PortLabelProps {
  ch: Channel;
  /** The entries of the slave in the direction of the task. */
  entries: ethercat.PDOEntry[];
}

/** Names the entry of a channel, then its address dimmed. */
export const PortLabel = ({ ch, entries }: PortLabelProps): ReactElement => {
  const port = getPortLabel(ch);
  const entry = hasEntry(ch) ? entryAt(entries, ch) : null;
  if (entry == null) return <>{port}</>;
  return (
    <Flex.Box el="span" x align="center" gap="small">
      {entry.name}{" "}
      <Text.Text level="small" color={7}>
        {port}
      </Text.Text>
    </Flex.Box>
  );
};
