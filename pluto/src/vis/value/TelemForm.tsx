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
import { deep, notation } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Channel } from "@/channel";
import { Notation } from "@/notation";
import { Staleness } from "@/vis/staleness";

type ValueTelemFormT = Pick<
  schematic.NumericTelemConfig,
  "channel" | "rollingAverage" | "precision" | "notation"
>;

const path = deep.path<ValueTelemFormT>();

/** TelemForm renders telemetry sections; place it inside `Form.Sections`. */
export const TelemForm = (): ReactElement => {
  const { value, onChange } = Form.useField(path);

  const handleSourceChange = (key: channel.Key | null): void =>
    onChange({ ...value, channel: key ?? 0 });

  const handleNotationChange = (notation: notation.Notation): void =>
    onChange({ ...value, notation });

  const handlePrecisionChange = (precision: number): void =>
    onChange({ ...value, precision });

  const handleRollingAverageChange = (windowSize: number): void =>
    onChange({ ...value, rollingAverage: windowSize });

  return (
    <>
      <Form.Section title="Source">
        <Input.Item label="Channel" padHelpText={false}>
          <Channel.SelectSingle
            value={value.channel}
            onChange={handleSourceChange}
            allowNone
          />
        </Input.Item>
        <Input.Item label="Averaging window" padHelpText={false}>
          <Input.Numeric
            value={value.rollingAverage}
            bounds={{ lower: 1, upper: 100 }}
            integer
            onChange={handleRollingAverageChange}
          />
        </Input.Item>
      </Form.Section>
      <Form.Section title="Format">
        <Input.Item label="Notation" padHelpText={false}>
          <Notation.Select value={value.notation} onChange={handleNotationChange} />
        </Input.Item>
        <Input.Item label="Precision" padHelpText={false}>
          <Input.Numeric
            value={value.precision ?? 2}
            onChange={handlePrecisionChange}
            integer
            bounds={notation.PRECISION_BOUNDS}
          />
        </Input.Item>
      </Form.Section>
      <Form.Section title="Staleness">
        <Staleness.Fields />
      </Form.Section>
    </>
  );
};
