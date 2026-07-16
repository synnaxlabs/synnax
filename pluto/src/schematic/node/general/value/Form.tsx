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

import { Flex } from "@/flex";
import { Form as Base } from "@/form";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Orientation } from "@/schematic/node/common/orientation";
import { Select } from "@/select";
import { Tabs } from "@/tabs";
import { Value } from "@/vis/value";

export const ValueForm = (): ReactElement => (
  <Tabs.Frame initialValue="style">
    <Tabs.Selector>
      <Tabs.Tab itemKey="style">Style</Tabs.Tab>
      <Tabs.Tab itemKey="telemetry">Telemetry</Tabs.Tab>
      <Tabs.Tab itemKey="redline">Redline</Tabs.Tab>
    </Tabs.Selector>
    <Tabs.Content itemKey="style">
      <Form.Wrapper x>
        <Flex.Box y grow>
          <Label.Form path="label" />
          <Flex.Box x>
            <Form.ColorField path="color" />
            <Form.UnitsField />
            <Base.NumericField
              path="inlineSize"
              label="Value width"
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
    </Tabs.Content>
    <Tabs.Content itemKey="telemetry">
      <Form.Wrapper y empty>
        <Value.TelemForm path="" />
      </Form.Wrapper>
    </Tabs.Content>
    <Tabs.Content itemKey="redline">
      <Form.Wrapper y empty>
        <Value.RedlineForm path="redline" />
      </Form.Wrapper>
    </Tabs.Content>
  </Tabs.Frame>
);
