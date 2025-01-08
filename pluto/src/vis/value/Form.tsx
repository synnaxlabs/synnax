// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { type ReactElement, useEffect } from "react";

import { Align } from "@/align";
import { Channel } from "@/channel";
import { telem } from "@/ether";
import { Form } from "@/form";
import { Input } from "@/input";
import { Notation } from "@/notation";

interface ValueTelemFormT {
  telem: telem.StringSourceSpec;
  tooltip: string[];
}

export const VALUE_CONNECTIONS: telem.Connection[] = [
  { from: "valueStream", to: "rollingAverage" },
  { from: "rollingAverage", to: "stopwatch" },
  { from: "stopwatch", to: "stringifier" },
];

interface TelemFormProps {
  path: string;
}

export const TelemForm = ({ path }: TelemFormProps): ReactElement => {
  const { value, onChange } = Form.useField<ValueTelemFormT>({ path });
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
        stopwatch: telem.stopwatch({}),
        rollingAverage: telem.rollingAverage({
          windowSize: rollingAverage.windowSize ?? 1,
        }),
        ...segments,
      },
      outlet: "stringifier",
    });
    onChange({ ...value, telem: t });
  };

  const handleSourceChange = (v: channel.Key | null): void =>
    handleChange({ valueStream: telem.streamChannelValue({ channel: v ?? 0 }) });

  const handleNotationChange = (notation: Notation.Notation): void =>
    handleChange({ stringifier: telem.stringifyNumber({ ...stringifier, notation }) });

  const handlePrecisionChange = (precision: number): void =>
    handleChange({ stringifier: telem.stringifyNumber({ ...stringifier, precision }) });

  const handleRollingAverageChange = (windowSize: number): void =>
    handleChange({ rollingAverage: telem.rollingAverage({ windowSize }) });

  const c = Channel.useName(source.channel as number);
  useEffect(() => onChange({ ...value, tooltip: [c] }), [c]);

  return (
    <>
      <Input.Item label="Input Channel" grow>
        <Channel.SelectSingle
          value={source.channel as number}
          onChange={handleSourceChange}
        />
      </Input.Item>
      <Align.Space direction="x">
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
      </Align.Space>
    </>
  );
};
