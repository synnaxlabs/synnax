// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { color, primitive, type text } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { Channel } from "@/channel";
import { Color } from "@/color";
import { Flex } from "@/flex";
import { Form as Base } from "@/form";
import { Input } from "@/input";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Orientation } from "@/schematic/node/common/orientation";
import { Select } from "@/select";
import { Tabs } from "@/tabs";
import { telem } from "@/telem/aether";

const STRING_DISPLAY_FORM_TABS: Tabs.Tab[] = [
  { tabKey: "style", name: "Style" },
  { tabKey: "telemetry", name: "Telemetry" },
];

const TelemForm = (): ReactElement => {
  const { set } = Base.useContext();
  const { value, onChange } = Base.useField<telem.StringSourceSpec>("telem");
  const source = telem.streamChannelValuePropsZ.parse(value?.props);
  const { retrieve } = Channel.useRetrieveObservable({
    onChange: useCallback(
      ({ data }) => data != null && set("tooltip", [data.name]),
      [set],
    ),
  });
  const handleSourceChange = (key: channel.Key | null): void => {
    if (primitive.isNonZero(key)) retrieve({ key });
    onChange(telem.streamChannelStringValue({ channel: key ?? 0 }));
  };
  if (typeof source.channel != "number")
    throw new Error("Must pass in a channel by key to the String Display form");
  return (
    <>
      <Input.Item label="Input channel" grow>
        <Channel.SelectSingle
          value={source.channel}
          onChange={handleSourceChange}
          // Only variable density channels (STRING, JSON, UUID) read as text.
          filter={(ch) => ch.dataType.isVariable}
        />
      </Input.Item>
      <Flex.Box x>
        <Base.Field<color.Crude>
          hideIfNull
          label="Stale color"
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
        </Base.Field>
        <Base.NumericField
          path="stalenessTimeout"
          label="Stale timeout"
          inputProps={{ bounds: { lower: 1, upper: Infinity }, endContent: "s" }}
        />
      </Flex.Box>
    </>
  );
};

const StyleForm = (): ReactElement => (
  <Form.Wrapper x>
    <Flex.Box y grow>
      <Label.Form path="label" />
      <Flex.Box x>
        <Form.ColorField path="color" />
        <Base.NumericField
          path="inlineSize"
          label="Display width"
          hideIfNull
          inputProps={Form.VALUE_WIDTH_INPUT_PROPS}
        />
        <Base.Field<text.Level>
          path="level"
          label="Size"
          hideIfNull
          padHelpText={false}
        >
          {({ value, onChange }) => (
            <Select.Text.Level value={value} onChange={onChange} />
          )}
        </Base.Field>
      </Flex.Box>
    </Flex.Box>
    <Orientation.Field path="" hideInner />
  </Form.Wrapper>
);

export const StringDisplayForm = (): ReactElement => {
  const content: Tabs.RenderProp = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "telemetry":
        return (
          <Form.Wrapper y empty>
            <TelemForm />
          </Form.Wrapper>
        );
      default:
        return <StyleForm />;
    }
  }, []);
  const props = Tabs.useStatic({ tabs: STRING_DISPLAY_FORM_TABS, content });
  return <Tabs.Tabs {...props} />;
};
