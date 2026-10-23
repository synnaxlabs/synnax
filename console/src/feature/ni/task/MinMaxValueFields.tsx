// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";

const MinValueField = Form.buildNumericField({
  fieldKey: "minVal",
  fieldProps: { label: "Minimum value" },
  inputProps: {},
});

const MaxValueField = Form.buildNumericField({
  fieldKey: "maxVal",
  fieldProps: { label: "Maximum value" },
  inputProps: {},
});

export interface MinMaxValueFieldsProps {
  path: string;
  units?: string;
}

export const MinMaxValueFields = ({ path, units }: MinMaxValueFieldsProps) => (
  <Input.Item label="Range">
    <Flex.Box x gap="small">
      <MinValueField
        path={path}
        showLabel={false}
        padHelpText={false}
        inputProps={{ endContent: units, grow: true, "aria-label": "Minimum value" }}
      />
      <MaxValueField
        path={path}
        showLabel={false}
        padHelpText={false}
        inputProps={{ endContent: units, grow: true, "aria-label": "Maximum value" }}
      />
    </Flex.Box>
  </Input.Item>
);
