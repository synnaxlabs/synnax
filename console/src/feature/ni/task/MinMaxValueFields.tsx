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
import { type deep } from "@synnaxlabs/x";

export interface MinMaxValueFieldsProps {
  path: deep.Path<{ minVal: number; maxVal: number }>;
  units?: string;
}

export const MinMaxValueFields = ({ path, units }: MinMaxValueFieldsProps) => {
  const min = Form.useFieldState(path.minVal);
  const max = Form.useFieldState(path.maxVal);
  // The inputs hide their help text, so the item shows the first problem.
  const problem = [min, max].find(({ status }) => status.variant !== "success")?.status;
  return (
    <Input.Item label="Range" helpText={problem?.message} status={problem?.variant}>
      <Flex.Box x gap="small">
        <Form.NumericField
          path={path.minVal}
          label="Minimum value"
          showLabel={false}
          showHelpText={false}
          inputProps={{ endContent: units, grow: true, "aria-label": "Minimum value" }}
        />
        <Form.NumericField
          path={path.maxVal}
          label="Maximum value"
          showLabel={false}
          showHelpText={false}
          inputProps={{ endContent: units, grow: true, "aria-label": "Maximum value" }}
        />
      </Flex.Box>
    </Input.Item>
  );
};
