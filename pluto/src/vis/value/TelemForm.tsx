// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { type color, type notation, primitive } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { Channel } from "@/channel";
import { Color } from "@/color";
import { Flex } from "@/flex";
import { Form } from "@/form";
import { Input } from "@/input";
import { Notation } from "@/notation";
import { Theming } from "@/theming";

interface ValueTelemFormT {
  channel?: channel.Key;
  rollingAverage?: number;
  precision?: number;
  notation?: notation.Notation;
}

export interface TelemFormProps {
  path: string;
}

export const TelemForm = ({ path }: TelemFormProps): ReactElement => {
  const { set } = Form.useContext();
  const { value, onChange } = Form.useField<ValueTelemFormT>(path);
  const theme = Theming.use();

  const { retrieve } = Channel.useRetrieveObservable({
    onChange: useCallback(
      ({ data }) => data != null && set(`${path}.tooltip`, [data.name]),
      [set, path],
    ),
  });
  const handleSourceChange = (key: channel.Key | null): void => {
    if (primitive.isNonZero(key)) retrieve({ key });
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
      <Input.Item label="Input channel" grow>
        <Channel.SelectSingle value={channelKey} onChange={handleSourceChange} />
      </Input.Item>
      <Flex.Box x>
        <Input.Item label="Notation">
          <Notation.Select
            value={value.notation ?? "standard"}
            onChange={handleNotationChange}
          />
        </Input.Item>
        <Input.Item label="Precision" align="start">
          <Input.Numeric
            value={value.precision ?? 2}
            bounds={{ lower: 0, upper: 10 }}
            onChange={handlePrecisionChange}
          />
        </Input.Item>
        <Input.Item label="Averaging window" align="start">
          <Input.Numeric
            value={value.rollingAverage ?? 1}
            bounds={{ lower: 1, upper: 100 }}
            onChange={handleRollingAverageChange}
          />
        </Input.Item>
        <Form.Field<color.Crude>
          hideIfNull={false}
          label="Stale color"
          align="start"
          path="stalenessColor"
        >
          {({ value, onChange }) => (
            <Color.Swatch
              value={value ?? theme.colors.warning.m1}
              onChange={onChange}
              bordered
            />
          )}
        </Form.Field>
        <Form.NumericField
          path="stalenessTimeout"
          label="Stale timeout"
          inputProps={{
            bounds: { lower: 1, upper: Infinity },
            endContent: "s",
          }}
        />
      </Flex.Box>
    </>
  );
};
