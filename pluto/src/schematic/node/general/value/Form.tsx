// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useCallback } from "react";

import { Flex } from "@/flex";
import { Form } from "@/form";
import { Input } from "@/input";
import {
  ColorControl,
  FormWrapper,
  LabelControls,
  OrientationControl,
  valueWidthInputProps,
} from "@/schematic/node/common/forms";
import { Select } from "@/select";
import { Tabs } from "@/tabs";
import { type Text } from "@/text";
import { Value } from "@/vis/value";

const VALUE_FORM_TABS: Tabs.Tab[] = [
  { tabKey: "style", name: "Style" },
  { tabKey: "telemetry", name: "Telemetry" },
  { tabKey: "redline", name: "Redline" },
];

export const ValueForm = (): ReactElement => {
  const content: Tabs.RenderProp = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "telemetry":
        return (
          <FormWrapper y empty>
            <Value.TelemForm path="" />
          </FormWrapper>
        );
      case "redline":
        return (
          <FormWrapper y empty>
            <Value.RedlineForm path="redline" />
          </FormWrapper>
        );
      default:
        return (
          <FormWrapper x>
            <Flex.Box y grow>
              <LabelControls path="label" />
              <Flex.Box x>
                <ColorControl path="color" />
                <Form.Field<string>
                  path="units"
                  label="Units"
                  align="start"
                  padHelpText={false}
                >
                  {(p) => <Input.Text {...p} />}
                </Form.Field>
                <Form.NumericField
                  path="inlineSize"
                  label="Value Width"
                  hideIfNull
                  inputProps={valueWidthInputProps}
                />
                <Form.Field<Text.Level>
                  path="level"
                  label="Size"
                  hideIfNull
                  padHelpText={false}
                >
                  {({ value, onChange }) => (
                    <Select.Text.Level value={value} onChange={onChange} />
                  )}
                </Form.Field>
              </Flex.Box>
            </Flex.Box>
            <OrientationControl path="" hideInner />
          </FormWrapper>
        );
    }
  }, []);
  const props = Tabs.useStatic({ tabs: VALUE_FORM_TABS, content });
  return <Tabs.Tabs {...props} />;
};
