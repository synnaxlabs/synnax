// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/textBox/textBox.css";

import { Button } from "@synnaxlabs/lyra/button";
import { CSS } from "@synnaxlabs/lyra/css";
import { type Flex } from "@synnaxlabs/lyra/flex";
import { Form as Base } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { type text } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Form } from "@/schematic/node/common/form";
import { Orientation } from "@/schematic/node/common/orientation";

const WRAP_WIDTH_INPUT_PROPS: Partial<Input.NumericProps> = {
  bounds: { lower: 0, upper: 2000 },
  dragScale: 5,
  endContent: "px",
};

export const TextBoxForm = (): ReactElement => {
  const autoFitDisabled = Base.useField<boolean>("autoFitDisabled");
  return (
    <Base.Sections x>
      <Base.Section title="Text">
        <Base.TextField path="value" label="Text" padHelpText={false} />
        <Base.Field<text.Level> path="level" label="Size" padHelpText={false}>
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
      </Base.Section>
      <Base.Section title="Appearance">
        <Form.ColorField path="color" />
        <Base.Field<number>
          onChange={(_, { set }) => set("autoFitDisabled", true)}
          path="width"
          label="Wrap width"
          padHelpText={false}
        >
          {(p) => (
            <Input.Numeric {...p} {...WRAP_WIDTH_INPUT_PROPS}>
              <Button.Button
                onClick={() => autoFitDisabled.onChange(false)}
                disabled={!autoFitDisabled.value}
                variant="outlined"
                className={CSS.BE("text-box-form", "auto-fit-btn")}
                tooltip={
                  !autoFitDisabled.value
                    ? "Manually enter value to disable auto fit"
                    : "Enable auto fit"
                }
              >
                <Icon.AutoFitWidth />
              </Button.Button>
            </Input.Numeric>
          )}
        </Base.Field>
      </Base.Section>
      <Orientation.Section path="" />
    </Base.Sections>
  );
};
