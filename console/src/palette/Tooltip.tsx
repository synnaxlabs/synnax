// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Align, Text, Triggers } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { type TriggerConfig } from "@/palette/types";

export interface TooltipContentProps {
  triggerConfig: TriggerConfig;
}

export const TooltipContent = ({
  triggerConfig: { command, search },
}: TooltipContentProps): ReactElement => (
  <Align.Space size="small">
    <Align.Space direction="x" justify="spaceBetween" align="center">
      <Text.Text level="small">Search</Text.Text>
      <Align.Space direction="x" empty size={0.5}>
        <Triggers.Text trigger={search[0]} level="small" />
      </Align.Space>
    </Align.Space>
    <Align.Space direction="x" justify="spaceBetween" align="center">
      <Text.Text level="small">Command Palette</Text.Text>
      <Align.Space direction="x" size={0.5}>
        <Triggers.Text trigger={command[0]} level="small" />
      </Align.Space>
    </Align.Space>
  </Align.Space>
);
