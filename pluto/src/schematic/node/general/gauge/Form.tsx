// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/charon/flex";
import type { Input } from "@synnaxlabs/charon/input";
import { Tabs } from "@synnaxlabs/charon/tabs";
import type { Text } from "@synnaxlabs/charon/text";
import { type ReactElement, useCallback } from "react";

import { Form as Base } from "@/form";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Select } from "@/select";
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

const handleLevelChange = (v: Text.Level, { set }: Base.ContextValue): void => {
  if (v === "small") set("barWidth", 4);
  else if (v === "h5") set("barWidth", 8);
  else set("barWidth", 10);
};

export const GaugeForm = (): ReactElement => {
  const content: Tabs.RenderProp = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "telemetry":
        return (
          <Form.Wrapper y empty>
            <Value.TelemForm path="" />
          </Form.Wrapper>
        );
      default:
        return (
          <Form.Wrapper x>
            <Flex.Box y grow>
              <Label.Form path="label" />
              <Flex.Box x>
                <Form.ColorField path="color" />
                <Base.TextField
                  path="units"
                  label="Units"
                  align="start"
                  padHelpText={false}
                />
                <Base.NumericField
                  path="bounds.lower"
                  label="Min Value"
                  hideIfNull
                  inputProps={BOUND_INPUT_PROPS}
                />
                <Base.NumericField
                  path="bounds.upper"
                  label="Max Value"
                  hideIfNull
                  inputProps={BOUND_INPUT_PROPS}
                />
                <Base.NumericField
                  path="barWidth"
                  label="Bar Width"
                  hideIfNull
                  inputProps={GAUGE_BAR_WIDTH_INPUT_PROPS}
                />
                <Base.Field<Text.Level>
                  path="level"
                  label="Size"
                  hideIfNull
                  padHelpText={false}
                  onChange={handleLevelChange}
                >
                  {({ value, onChange }) => (
                    <Select.Text.Level value={value} onChange={onChange} />
                  )}
                </Base.Field>
              </Flex.Box>
            </Flex.Box>
          </Form.Wrapper>
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
