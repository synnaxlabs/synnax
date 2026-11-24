// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { channel } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { DataType, deep } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { Channel } from "@/channel";

const path = deep.path<{ stateChannel?: channel.Key }>();

/** Lists the channels of variable density, which read as text. */
export const VARIABLE_CHANNELS: Channel.ListQuery = {
  where: { dataType: { in: [DataType.STRING, DataType.JSON, DataType.BYTES] } },
};

export interface StateSectionProps {
  /** The channels the select offers. */
  query?: Channel.ListQuery;
}

/** StateSection edits the channel a control reads the device's value back from. */
export const StateSection = ({ query }: StateSectionProps): ReactElement => {
  const { value, onChange } = Form.useField(path);
  return (
    <Form.Section title="State">
      <Input.Item label="Channel" padHelpText={false}>
        <Channel.SelectSingle
          allowNone
          value={value.stateChannel}
          onChange={(v: channel.Key | null) =>
            onChange({ ...value, stateChannel: v ?? 0 })
          }
          query={query}
        />
      </Input.Item>
    </Form.Section>
  );
};
