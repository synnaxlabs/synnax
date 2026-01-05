// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { color, type notation, primitive } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { Channel } from "@/channel";
import { Color } from "@/color";
import { telem } from "@/ether";
import { Flex } from "@/flex";
import { Form } from "@/form";
import { Input } from "@/input";
import { Notation } from "@/notation";

interface ValueTelemFormT {
  telem: telem.StringSourceSpec;
  tooltip: string[];
  stalenessTimeout?: number;
  stalenessColor?: color.Color;
}

const VALUE_CONNECTIONS: telem.Connection[] = [
  { from: "valueStream", to: "rollingAverage" },
  { from: "rollingAverage", to: "stringifier" },
];

export interface TelemFormProps {
  path: string;
}

export const TelemForm = ({ path }: TelemFormProps): ReactElement => {
  const { set } = Form.useContext();
  const { value, onChange } = Form.useField<ValueTelemFormT>(path);
  const sourceP = telem.sourcePipelinePropsZ.parse(value.telem?.props);
  const source = telem.streamChannelValuePropsZ.parse(
    sourceP.segments.valueStream.props,
  );
  const stringifier = telem.stringifyNumberProps.parse(
    sourceP.segments.stringifier.props,
  );
  const rollingAverage = telem.rollingAverageProps.parse(
    sourceP.segments.rollingAverage.props,
  );

  const handleChange = (segments: telem.SourcePipelineProps["segments"]): void => {
    const t = telem.sourcePipeline("string", {
      connections: VALUE_CONNECTIONS,
      segments: {
        valueStream: telem.streamChannelValue({ channel: source.channel }),
        stringifier: telem.stringifyNumber({
          precision: stringifier.precision ?? 2,
          notation: stringifier.notation,
        }),
        rollingAverage: telem.rollingAverage({
          windowSize: rollingAverage.windowSize ?? 1,
        }),
        ...segments,
      },
      outlet: "stringifier",
    });
    onChange({ ...value, telem: t });
  };

  const { retrieve } = Channel.useRetrieveObservable({
    onChange: useCallback(
      ({ data }) => data != null && set(`${path}.tooltip`, [data.name]),
      [set, path],
    ),
  });
  const handleSourceChange = (key: channel.Key | null): void => {
    if (primitive.isNonZero(key)) retrieve({ key });
    handleChange({ valueStream: telem.streamChannelValue({ channel: key ?? 0 }) });
  };

  const handleNotationChange = (notation: notation.Notation): void =>
    handleChange({ stringifier: telem.stringifyNumber({ ...stringifier, notation }) });

  const handlePrecisionChange = (precision: number): void =>
    handleChange({ stringifier: telem.stringifyNumber({ ...stringifier, precision }) });

  const handleRollingAverageChange = (windowSize: number): void =>
    handleChange({ rollingAverage: telem.rollingAverage({ windowSize }) });

  if (typeof source.channel != "number")
    throw new Error("Must pass in a channel by key to Value.TelemForm");
  const channelKey = source.channel;

  return (
    <>
      <Input.Item label="Input Channel" grow>
        <Channel.SelectSingle value={channelKey} onChange={handleSourceChange} />
      </Input.Item>
      <Flex.Box x>
        <Input.Item label="Notation">
          <Notation.Select
            value={stringifier.notation}
            onChange={handleNotationChange}
          />
        </Input.Item>
        <Input.Item label="Precision" align="start">
          <Input.Numeric
            value={stringifier.precision ?? 2}
            bounds={{ lower: 0, upper: 10 }}
            onChange={handlePrecisionChange}
          />
        </Input.Item>
        <Input.Item label="Averaging Window" align="start">
          <Input.Numeric
            value={rollingAverage.windowSize ?? 1}
            bounds={{ lower: 1, upper: 100 }}
            onChange={handleRollingAverageChange}
          />
        </Input.Item>
        <Form.Field<color.Crude>
          hideIfNull
          label="Stale Color"
          align="start"
          path="stalenessColor"
        >
          {({ value, onChange }) => (
            <Color.Swatch
              value={value ?? color.setAlpha(color.ZERO, 1)}
              onChange={onChange}
              bordered
            />
          )}
        </Form.Field>
        <Form.NumericField
          path="stalenessTimeout"
          label="Stale Timeout"
          inputProps={{
            bounds: { lower: 1, upper: Infinity },
            endContent: "s",
          }}
        />
      </Flex.Box>
    </>
  );
};
