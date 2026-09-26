// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type schematic } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { type ReactElement } from "react";

import { Channel } from "@/channel";
import { Control } from "@/schematic/node/common/control";
import { ActivationDelayField } from "@/schematic/node/common/form/ActivationDelay";
import { ControlChipField } from "@/schematic/node/common/form/Control";
import { Staleness } from "@/vis/staleness";

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
      control: Control.reveal(value.control),
    });

  return (
    <Form.Sections x>
      <Form.Section title="State">
        <Input.Item label="Channel" padHelpText={false}>
          <Channel.SelectSingle
            value={value.stateChannel ?? 0}
            onChange={handleSourceChange}
          />
        </Input.Item>
      </Form.Section>
      <Form.Section title="Command">
        <Input.Item label="Channel" padHelpText={false}>
          <Channel.SelectSingle
            value={value.commandChannel ?? 0}
            onChange={handleSinkChange}
          />
        </Input.Item>
        {!omit.includes("onClickDelay") && <ActivationDelayField />}
        <ControlChipField />
      </Form.Section>
      <Form.Section title="Staleness">
        <Staleness.Fields />
      </Form.Section>
    </Form.Sections>
  );
};
