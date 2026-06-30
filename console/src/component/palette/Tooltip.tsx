// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Text, Triggers } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { type TriggerConfig } from "@/palette/types";

export interface TooltipContentProps {
  triggerConfig: TriggerConfig;
}

export const TooltipContent = ({
  triggerConfig: { command, search },
}: TooltipContentProps): ReactElement => (
  <Flex.Box gap="small">
    <Flex.Box x justify="between" align="center">
      <Text.Text level="small">Search</Text.Text>
      <Flex.Box x empty gap="tiny">
        <Triggers.Text trigger={search[0]} level="small" />
      </Flex.Box>
    </Flex.Box>
    <Flex.Box x justify="between" align="center">
      <Text.Text level="small">Command Palette</Text.Text>
      <Flex.Box x gap="tiny">
        <Triggers.Text trigger={command[0]} level="small" />
      </Flex.Box>
    </Flex.Box>
  </Flex.Box>
);
