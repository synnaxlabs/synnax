// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type schematic } from "@synnaxlabs/client";
import { Form as Base } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Status } from "@synnaxlabs/lyra/status";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { primitive } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Channel } from "@/channel";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Orientation } from "@/schematic/node/common/orientation";
import { Synnax } from "@/synnax";
import { Staleness } from "@/vis/staleness";

const TelemForm = (): ReactElement => {
  const { set } = Base.useContext();
  const { value, onChange } =
    Base.useField<Pick<schematic.StringDisplayNodeConfig, "channel">>("");
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const handleSourceChange = (key: channel.Key | null): void => {
    if (primitive.isNonZero(key) && client != null)
      handleError(async () => {
        const { name } = await client.channels.retrieve({ key });
        set("tooltip", [name]);
      }, "Failed to retrieve channel");
    onChange({ ...value, channel: key ?? undefined });
  };
  return (
    <Base.Sections x>
      <Base.Section title="Source">
        <Input.Item label="Channel" padHelpText={false}>
          <Channel.SelectSingle
            value={value.channel ?? 0}
            onChange={handleSourceChange}
            // Only variable density channels (STRING, JSON, UUID) read as text.
            filter={(ch) => ch.dataType.isVariable}
          />
        </Input.Item>
      </Base.Section>
      <Base.Section title="Staleness">
        <Staleness.Fields />
      </Base.Section>
    </Base.Sections>
  );
};

const StyleForm = (): ReactElement => (
  <Base.Sections x>
    <Base.Section title="Label">
      <Label.Form path="label" />
    </Base.Section>
    <Base.Section title="Appearance">
      <Form.ColorField path="color" />
      <Form.LevelSizeField />
      <Base.NumericField
        path="inlineSize"
        label="Width"
        hideIfNull
        padHelpText={false}
        inputProps={Form.VALUE_WIDTH_INPUT_PROPS}
      />
    </Base.Section>
    <Orientation.Section path="" hideInner />
  </Base.Sections>
);

export const StringDisplayForm = (): ReactElement => (
  <Form.Tabs tabs={["style", "telemetry"]}>
    <Tabs.Content itemKey="style">
      <StyleForm />
    </Tabs.Content>
    <Tabs.Content itemKey="telemetry">
      <TelemForm />
    </Tabs.Content>
  </Form.Tabs>
);
