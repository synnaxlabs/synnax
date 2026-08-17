// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { primitive, type text, zod } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Channel } from "@/channel";
import { Flex } from "@/flex";
import { Form as Base } from "@/form";
import { Input } from "@/input";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Orientation } from "@/schematic/node/common/orientation";
import { Select } from "@/select";
import { Status } from "@/status";
import { Synnax } from "@/synnax";
import { Tabs } from "@/tabs";
import { telem } from "@/telem/aether";
import { Staleness } from "@/vis/staleness";

const TelemForm = (): ReactElement => {
  const { set } = Base.useContext();
  const { value, onChange } = Base.useField<telem.StringSourceSpec>("telem");
  const source = zod.parse(telem.streamChannelValuePropsZ, value?.props, {
    label: "value stream source",
  });
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const handleSourceChange = (key: channel.Key | null): void => {
    if (primitive.isNonZero(key) && client != null)
      handleError(async () => {
        const { name } = await client.channels.retrieve({ key });
        set("tooltip", [name]);
      }, "Failed to retrieve channel");
    onChange(telem.streamChannelStringValue({ channel: key ?? 0 }));
  };
  if (typeof source.channel != "number")
    throw new Error("Must pass in a channel by key to the String Display form");
  return (
    <>
      <Input.Item label="Channel" grow>
        <Channel.SelectSingle
          value={source.channel}
          onChange={handleSourceChange}
          // Only variable density channels (STRING, JSON, UUID) read as text.
          filter={(ch) => ch.dataType.isVariable}
        />
      </Input.Item>
      <Flex.Box x>
        <Staleness.Fields />
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

export const StringDisplayForm = (): ReactElement => (
  <Tabs.Frame initialValue="style">
    <Tabs.Selector>
      <Tabs.Tab itemKey="style">Style</Tabs.Tab>
      <Tabs.Tab itemKey="telemetry">Telemetry</Tabs.Tab>
    </Tabs.Selector>
    <Tabs.Content itemKey="style">
      <StyleForm />
    </Tabs.Content>
    <Tabs.Content itemKey="telemetry">
      <Form.Wrapper y empty>
        <TelemForm />
      </Form.Wrapper>
    </Tabs.Content>
  </Tabs.Frame>
);
