// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type text } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { Component } from "@/component";
import { Flex } from "@/flex";
import { Form as Base } from "@/form";
import { type Input } from "@/input";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { type Side } from "@/schematic/node/general/scale/config";
import { Select } from "@/select";
import { Tabs } from "@/tabs";
import { type scale } from "@/vis/scale/aether";
import { Value } from "@/vis/value";

const BOUND_INPUT_PROPS: Partial<Input.NumericProps> = { step: 10 };

const STYLE_KEYS = ["fill", "caret"] as const;
const StyleSelect = Component.renderProp(
  ({
    value,
    onChange,
  }: {
    value?: scale.Style;
    onChange: (v: scale.Style) => void;
  }): ReactElement => (
    <Select.Buttons value={value ?? "fill"} onChange={onChange} keys={STYLE_KEYS}>
      <Select.Button itemKey="fill">Fill</Select.Button>
      <Select.Button itemKey="caret">Caret</Select.Button>
    </Select.Buttons>
  ),
);

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

export const ScaleForm = (): ReactElement => {
  const content: Tabs.RenderProp = useCallback(({ tabKey }) => {
    if (tabKey === "telemetry")
      return (
        <Form.Wrapper y empty>
          <Value.TelemForm path="" />
        </Form.Wrapper>
      );
    return (
      <Form.Wrapper x>
        <Flex.Box y grow>
          <Label.Form path="label" />
          <Flex.Box x>
            <Form.ColorField path="color" />
            <Base.NumericField
              path="bounds.lower"
              label="Min value"
              hideIfNull
              inputProps={BOUND_INPUT_PROPS}
            />
            <Base.NumericField
              path="bounds.upper"
              label="Max value"
              hideIfNull
              inputProps={BOUND_INPUT_PROPS}
            />
            <Base.Field<scale.Style> path="style" label="Indicator" padHelpText={false}>
              {StyleSelect}
            </Base.Field>
            <Base.Field<Side> path="side" label="Scale side" padHelpText={false}>
              {SideSelect}
            </Base.Field>
            <Base.Field<text.Level>
              path="level"
              label="Size"
              hideIfNull
              padHelpText={false}
            >
              {Form.SelectTextLevel}
            </Base.Field>
          </Flex.Box>
        </Flex.Box>
      </Form.Wrapper>
    );
  }, []);
  const tabs: Tabs.Spec[] = [
    { tabKey: "properties", name: "Properties" },
    { tabKey: "telemetry", name: "Telemetry" },
  ];
  const props = Tabs.useStatic({ tabs, content });
  return <Tabs.Tabs {...props} />;
};
