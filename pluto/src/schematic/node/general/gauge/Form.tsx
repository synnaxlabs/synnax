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
import { type Input } from "@/input";
import {
  ColorControl,
  FormWrapper,
  LabelControls,
} from "@/schematic/node/common/forms";
import { Select } from "@/select";
import { Tabs } from "@/tabs";
import { type Text } from "@/text";
import { Value } from "@/vis/value";

const GAUGE_BAR_WIDTH_INPUT_PROPS: Partial<Input.NumericProps> = {
  min: 1,
  max: 50,
  step: 1,
  bounds: { lower: 1, upper: 50 },
  endContent: "px",
  dragScale: { x: 0.1, y: 0.1 },
};

const BOUND_INPUT_PROPS: Partial<Input.NumericProps> = { step: 10 };

const handleLevelChange = (v: Text.Level, { set }: Form.ContextValue): void => {
  if (v === "small") set("barWidth", 4);
  else if (v === "h5") set("barWidth", 8);
  else set("barWidth", 10);
};

export const GaugeForm = (): ReactElement => {
  const content: Tabs.RenderProp = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "telemetry":
        return (
          <FormWrapper y empty>
            <Value.TelemForm path="" />
          </FormWrapper>
        );
      default:
        return (
          <FormWrapper x>
            <Flex.Box y grow>
              <LabelControls path="label" />
              <Flex.Box x>
                <ColorControl path="color" />
                <Form.TextField
                  path="units"
                  label="Units"
                  align="start"
                  padHelpText={false}
                />
                <Form.NumericField
                  path="bounds.lower"
                  label="Min Value"
                  hideIfNull
                  inputProps={BOUND_INPUT_PROPS}
                />
                <Form.NumericField
                  path="bounds.upper"
                  label="Max Value"
                  hideIfNull
                  inputProps={BOUND_INPUT_PROPS}
                />
                <Form.NumericField
                  path="barWidth"
                  label="Bar Width"
                  hideIfNull
                  inputProps={GAUGE_BAR_WIDTH_INPUT_PROPS}
                />
                <Form.Field<Text.Level>
                  path="level"
                  label="Size"
                  hideIfNull
                  padHelpText={false}
                  onChange={handleLevelChange}
                >
                  {({ value, onChange }) => (
                    <Select.Text.Level value={value} onChange={onChange} />
                  )}
                </Form.Field>
              </Flex.Box>
            </Flex.Box>
          </FormWrapper>
        );
    }
  }, []);
  const tabs: Tabs.Spec[] = [
    { tabKey: "properties", name: "Properties" },
    { tabKey: "telemetry", name: "Telemetry" },
  ];
  const props = Tabs.useStatic({ tabs, content });
  return <Tabs.Tabs {...props} />;
};
