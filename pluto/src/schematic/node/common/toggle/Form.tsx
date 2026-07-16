// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type schematic } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Channel } from "@/channel";
import { Flex } from "@/flex";
import { Form } from "@/form";
import { Input } from "@/input";
import { ActivationDelayField } from "@/schematic/node/common/form/ActivationDelay";
import { ControlChipField } from "@/schematic/node/common/form/Control";
import { Wrapper } from "@/schematic/node/common/form/Wrapper";

interface ChannelFormProps {
  path: string;
  omit?: string[];
}

export const ChannelForm = ({ path, omit = [] }: ChannelFormProps): ReactElement => {
  const { value, onChange } =
    Form.useField<
      Pick<schematic.ToggleConfig, "stateChannel" | "commandChannel" | "control">
    >(path);

  const handleSourceChange = (v: channel.Key | null): void =>
    onChange({ ...value, stateChannel: v ?? undefined });

  const handleSinkChange = (v: channel.Key | null): void =>
    onChange({
      ...value,
      commandChannel: v ?? undefined,
      control: { showChip: true, showIndicator: true, ...value.control },
    });

  return (
    <Wrapper y empty>
      <Flex.Box x grow>
        <Input.Item label="State channel" grow padHelpText={false}>
          <Channel.SelectSingle
            value={value.stateChannel ?? 0}
            onChange={handleSourceChange}
          />
        </Input.Item>
        <Input.Item label="Command channel" grow padHelpText={false}>
          <Channel.SelectSingle
            value={value.commandChannel ?? 0}
            onChange={handleSinkChange}
          />
        </Input.Item>
      </Flex.Box>
      <Flex.Box x grow>
        {!omit.includes("onClickDelay") && <ActivationDelayField grow />}
        <ControlChipField />
      </Flex.Box>
    </Wrapper>
  );
};
