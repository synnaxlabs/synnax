// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/textBox/textBox.css";

import { type text } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Form as Base } from "@/form";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { Form } from "@/schematic/node/common/form";
import { Orientation } from "@/schematic/node/common/orientation";
import { Select } from "@/select";

const WRAP_WIDTH_INPUT_PROPS: Partial<Input.NumericProps> = {
  bounds: { lower: 0, upper: 2000 },
  dragScale: 5,
  endContent: "px",
};

export const TextBoxForm = (): ReactElement => {
  const autoFit = Base.useField<boolean>("autoFit", { optional: true });
  return (
    <Form.Wrapper x align="stretch" grow>
      <Flex.Box y grow>
        <Flex.Box x align="stretch">
          <Base.TextField path="value" label="Text" padHelpText={false} grow />
          <Base.Field<text.Level> path="level" label="Text Size" padHelpText={false}>
            {({ value, onChange }) => (
              <Select.Text.Level value={value} onChange={onChange} />
            )}
          </Base.Field>
          <Base.Field<Flex.Alignment>
            path="align"
            label="Alignment"
            padHelpText={false}
            hideIfNull
          >
            {({ value, onChange }) => (
              <Select.Flex.Alignment value={value} onChange={onChange} />
            )}
          </Base.Field>
        </Flex.Box>
        <Flex.Box x>
          <Form.ColorField path="color" />
          <Base.Field<number>
            onChange={(_, { set }) => set("autoFit", false)}
            path="width"
            label="Wrap width"
            padHelpText={false}
          >
            {(p) => (
              <Input.Numeric {...p} {...WRAP_WIDTH_INPUT_PROPS}>
                <Button.Button
                  onClick={() => autoFit?.onChange(true)}
                  disabled={autoFit?.value === true}
                  variant="outlined"
                  className={CSS.BE("text-box-form", "auto-fit-btn")}
                  tooltip={
                    autoFit?.value === true
                      ? "Manually enter value to disable auto fit"
                      : "Enable auto fit"
                  }
                >
                  <Icon.AutoFitWidth />
                </Button.Button>
              </Input.Numeric>
            )}
          </Base.Field>
        </Flex.Box>
      </Flex.Box>
      <Orientation.Field path="" />
    </Form.Wrapper>
  );
};
