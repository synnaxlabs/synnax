// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type text } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Form as Base } from "@/form";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Orientation } from "@/schematic/node/common/orientation";
import { Select } from "@/select";
import { Tabs } from "@/tabs";
import { Value } from "@/vis/value";

export const ValueForm = (): ReactElement => (
  <Form.Tabs tabs={["style", "telemetry", "redline"]}>
    <Tabs.Content itemKey="style">
      <Base.Sections x>
        <Base.Section title="Label">
          <Label.Form path="label" />
        </Base.Section>
        <Base.Section title="Appearance">
          <Form.ColorField path="color" />
          <Form.UnitsField />
          <Base.NumericField
            path="inlineSize"
            label="Width"
            hideIfNull
            padHelpText={false}
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
        </Base.Section>
        <Orientation.Section path="" hideInner />
      </Base.Sections>
    </Tabs.Content>
    <Tabs.Content itemKey="telemetry">
      <Base.Sections x>
        <Value.TelemForm path="" />
      </Base.Sections>
    </Tabs.Content>
    <Tabs.Content itemKey="redline">
      <Base.Sections x>
        <Base.Section title="Redline">
          <Value.RedlineForm path="redline" />
        </Base.Section>
      </Base.Sections>
    </Tabs.Content>
  </Form.Tabs>
);
