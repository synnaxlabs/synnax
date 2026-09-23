// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { type notation, primitive } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Channel } from "@/channel";
import { Form } from "@/form";
import { Input } from "@/input";
import { Notation } from "@/notation";
import { Status } from "@/status";
import { Synnax } from "@/synnax";
import { Staleness } from "@/vis/staleness";

interface ValueTelemFormT {
  channel?: channel.Key;
  rollingAverage?: number;
  precision?: number;
  notation?: notation.Notation;
}

export interface TelemFormProps {
  path: string;
}

/** TelemForm is a value's source and format sections; the caller lays them out. */
export const TelemForm = ({ path }: TelemFormProps): ReactElement => {
  const { set } = Form.useContext();
  const { value, onChange } = Form.useField<ValueTelemFormT>(path);

  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const handleSourceChange = (key: channel.Key | null): void => {
    if (primitive.isNonZero(key) && client != null)
      handleError(async () => {
        const { name } = await client.channels.retrieve({ key });
        set(`${path}.tooltip`, [name]);
      }, "Failed to retrieve channel");
    onChange({ ...value, channel: key ?? undefined });
  };

  const handleNotationChange = (notation: notation.Notation): void =>
    onChange({ ...value, notation });

  const handlePrecisionChange = (precision: number): void =>
    onChange({ ...value, precision });

  const handleRollingAverageChange = (windowSize: number): void =>
    onChange({ ...value, rollingAverage: windowSize });

  const channelKey = value.channel ?? 0;

  return (
    <>
      <Form.Section title="Source">
        <Input.Item label="Channel" padHelpText={false}>
          <Channel.SelectSingle value={channelKey} onChange={handleSourceChange} />
        </Input.Item>
        <Input.Item label="Averaging window" padHelpText={false}>
          <Input.Numeric
            value={value.rollingAverage ?? 1}
            bounds={{ lower: 1, upper: 100 }}
            onChange={handleRollingAverageChange}
          />
        </Input.Item>
      </Form.Section>
      <Form.Section title="Format">
        <Input.Item label="Notation" padHelpText={false}>
          <Notation.Select
            value={value.notation ?? "standard"}
            onChange={handleNotationChange}
          />
        </Input.Item>
        <Input.Item label="Precision" padHelpText={false}>
          <Input.Numeric
            value={value.precision ?? 2}
            bounds={{ lower: 0, upper: 10 }}
            onChange={handlePrecisionChange}
          />
        </Input.Item>
      </Form.Section>
      <Form.Section title="Staleness">
        <Staleness.Fields />
      </Form.Section>
    </>
  );
};
