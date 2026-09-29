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
import { type bounds, type notation } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { Channel } from "@/channel";
import { Notation } from "@/notation";
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

// Stands in for an absent precision, which shows as many decimals as fit.
const AUTO_PRECISION = -1;
const PRECISION_BOUNDS: bounds.Bounds = { lower: 0, upper: 20 };
const ROLLING_AVERAGE_BOUNDS: bounds.Bounds = { lower: 1, upper: 100 };

/** TelemForm renders telemetry sections; place it inside `Form.Sections`. */
export const TelemForm = ({ path }: TelemFormProps): ReactElement => {
  const { value, onChange } = Form.useField<ValueTelemFormT>(path);

  const handleSourceChange = useCallback(
    (key: channel.Key | null) => onChange({ ...value, channel: key ?? undefined }),
    [value, onChange],
  );
  const handleNotationChange = useCallback(
    (notation: notation.Notation) => onChange({ ...value, notation }),
    [value, onChange],
  );
  const handlePrecisionChange = useCallback(
    (precision: number) =>
      onChange({
        ...value,
        precision: precision === AUTO_PRECISION ? undefined : precision,
      }),
    [value, onChange],
  );
  const handleRollingAverageChange = useCallback(
    (rollingAverage: number) => onChange({ ...value, rollingAverage }),
    [value, onChange],
  );

  return (
    <>
      <Form.Section title="Source">
        <Input.Item label="Channel" padHelpText={false}>
          <Channel.SelectSingle
            value={value.channel ?? 0}
            onChange={handleSourceChange}
          />
        </Input.Item>
        <Input.Item label="Averaging window" padHelpText={false}>
          <Input.Numeric
            value={value.rollingAverage ?? 1}
            bounds={ROLLING_AVERAGE_BOUNDS}
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
            value={value.precision ?? AUTO_PRECISION}
            emptyValue={AUTO_PRECISION}
            placeholder="Auto"
            bounds={PRECISION_BOUNDS}
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
