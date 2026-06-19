// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { primitive } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { Channel } from "@/channel";
import { Component } from "@/component";
import { Flex } from "@/flex";
import { Form as Base } from "@/form";
import { Input } from "@/input";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Orientation } from "@/schematic/node/common/orientation";
import { type FormProps as NodeFormProps } from "@/schematic/node/spec";
import { type Side } from "@/schematic/node/vessels/tank/config";
import { Select } from "@/select";
import { Tabs } from "@/tabs";
import { telem } from "@/telem/aether";

export interface TankFormProps extends NodeFormProps {
  showBorderRadius?: boolean;
  showStrokeWidth?: boolean;
  showFill?: boolean;
}

const BOUND_INPUT_PROPS = { step: 10 };

const FILL_CONNECTIONS: telem.Connection[] = [
  { from: "valueStream", to: "rollingAverage" },
  { from: "rollingAverage", to: "stringifier" },
];

const buildFillTelem = (channel: number): telem.StringSourceSpec =>
  telem.sourcePipeline("string", {
    connections: FILL_CONNECTIONS,
    segments: {
      valueStream: telem.streamChannelValue({ channel }),
      rollingAverage: telem.rollingAverage({ windowSize: 1 }),
      stringifier: telem.stringifyNumber({ precision: 2, notation: "standard" }),
    },
    outlet: "stringifier",
  });

const SIDE_KEYS = ["left", "right"] as const;
const SideSelect = Component.renderProp(
  ({
    value,
    onChange,
  }: {
    value?: Side;
    onChange: (v: Side) => void;
  }): ReactElement => (
    <Select.Buttons value={value ?? "right"} onChange={onChange} keys={SIDE_KEYS}>
      <Select.Button itemKey="left">Left</Select.Button>
      <Select.Button itemKey="right">Right</Select.Button>
    </Select.Buttons>
  ),
);

const FillForm = (): ReactElement => {
  const ctx = Base.useContext();
  const telemField = Base.useField<telem.StringSourceSpec>("telem", { optional: true });
  let channelKey = 0;
  const spec = telemField?.value;
  if (spec != null) {
    const source = telem.sourcePipelinePropsZ.parse(spec.props);
    channelKey = Number(
      telem.streamChannelValuePropsZ.parse(source.segments.valueStream.props).channel,
    );
  }
  const handleChannelChange = (key: channel.Key | null): void =>
    ctx.set(
      "telem",
      key == null || !primitive.isNonZero(key) ? undefined : buildFillTelem(key),
    );
  const enabled = primitive.isNonZero(channelKey);
  return (
    <Form.Wrapper y empty>
      <Input.Item label="Fill channel" grow>
        <Channel.SelectSingle
          value={channelKey}
          onChange={handleChannelChange}
          allowNone
        />
      </Input.Item>
      {enabled && (
        <Flex.Box x>
          <Form.ColorField path="fillColor" label="Fill color" />
          <Base.NumericField
            path="bounds.lower"
            label="Min value"
            inputProps={BOUND_INPUT_PROPS}
          />
          <Base.NumericField
            path="bounds.upper"
            label="Max value"
            inputProps={BOUND_INPUT_PROPS}
          />
          <Base.SwitchField path="showScale" label="Show scale" />
          <Base.Field<Side> path="scaleSide" label="Scale side" padHelpText={false}>
            {SideSelect}
          </Base.Field>
        </Flex.Box>
      )}
    </Form.Wrapper>
  );
};

export const TankForm = ({
  showBorderRadius = false,
  showStrokeWidth = false,
  showFill = false,
}: TankFormProps): ReactElement => {
  const properties = (
    <Form.Wrapper x align="stretch">
      <Flex.Box y grow>
        <Label.Form path="label" />
        <Flex.Box x>
          <Form.ColorField path="color" />
          <Form.ColorField path="backgroundColor" label="Background color" />
          <Base.NumericField
            path="borderRadius.x"
            hideIfNull
            optional
            label="X border radius"
            grow
            inputProps={Form.PERCENT_BORDER_RADIUS_INPUT_PROPS}
          />
          <Base.NumericField
            path="borderRadius.y"
            hideIfNull
            optional
            label="Y border radius"
            grow
            inputProps={Form.PERCENT_BORDER_RADIUS_INPUT_PROPS}
          />
          {showBorderRadius && (
            <Base.NumericField
              path="borderRadius"
              hideIfNull
              optional
              label="Border radius"
              grow
              inputProps={Form.DIMENSIONS_INPUT_PROPS}
            />
          )}
          {showStrokeWidth && (
            <Base.NumericField
              path="strokeWidth"
              hideIfNull
              optional
              label="Border width"
              grow
              inputProps={Form.STROKE_WIDTH_INPUT_PROPS}
            />
          )}
          <Base.NumericField
            path="dimensions.width"
            label="Width"
            grow
            inputProps={Form.DIMENSIONS_INPUT_PROPS}
          />
          <Base.NumericField
            path="dimensions.height"
            label="Height"
            grow
            inputProps={Form.DIMENSIONS_INPUT_PROPS}
          />
        </Flex.Box>
      </Flex.Box>
      <Orientation.Field path="" hideInner showOuterCenter label="Label location" />
    </Form.Wrapper>
  );
  const content: Tabs.RenderProp = useCallback(
    ({ tabKey }) => (tabKey === "fill" ? <FillForm /> : properties),
    [properties],
  );
  const tabsProps = Tabs.useStatic({
    tabs: [
      { tabKey: "properties", name: "Properties" },
      { tabKey: "fill", name: "Fill" },
    ],
    content,
  });
  if (!showFill) return properties;
  return <Tabs.Tabs {...tabsProps} />;
};
