// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as Base } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import type { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, useCallback } from "react";

import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Orientation } from "@/schematic/node/common/orientation";
import { Value } from "@/value";

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
          <Form.Wrapper y empty>
            <Value.TelemForm path="" />
          </Form.Wrapper>
        );
      case "redline":
        return (
          <Form.Wrapper y empty>
            <Value.RedlineForm path="redline" />
          </Form.Wrapper>
        );
      default:
        return (
          <Form.Wrapper x>
            <Flex.Box y grow>
              <Label.Form path="label" />
              <Flex.Box x>
                <Form.ColorField path="color" />
                <Base.Field<string>
                  path="units"
                  label="Units"
                  align="start"
                  padHelpText={false}
                >
                  {(p) => <Input.Text {...p} />}
                </Base.Field>
                <Base.NumericField
                  path="inlineSize"
                  label="Value Width"
                  hideIfNull
                  inputProps={Form.VALUE_WIDTH_INPUT_PROPS}
                />
                <Base.Field<Text.Level>
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
    }
  }, []);
  const props = Tabs.useStatic({ tabs: VALUE_FORM_TABS, content });
  return <Tabs.Tabs {...props} />;
};
