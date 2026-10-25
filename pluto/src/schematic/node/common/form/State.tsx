// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { type ReactElement } from "react";

import { Channel } from "@/channel";

export interface StateSectionProps {
  /** filter limits the channels the select offers. */
  filter?: (ch: channel.Channel) => boolean;
}

/** StateSection edits the channel a control reads the device's value back from. */
export const StateSection = ({ filter }: StateSectionProps): ReactElement => {
  const { value, onChange } = Form.useField<{ stateChannel?: channel.Key }>("");
  return (
    <Form.Section title="State">
      <Input.Item label="Channel" padHelpText={false}>
        <Channel.SelectSingle
          allowNone
          value={value.stateChannel ?? 0}
          onChange={(v: channel.Key | null) =>
            onChange({ ...value, stateChannel: v ?? undefined })
          }
          filter={filter}
        />
      </Input.Item>
    </Form.Section>
  );
};
