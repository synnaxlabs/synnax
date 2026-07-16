// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Form } from "@synnaxlabs/pluto";

const MinValueField = Form.buildNumericField({
  fieldKey: "minVal",
  fieldProps: { label: "Minimum Value" },
  inputProps: {},
});

const MaxValueField = Form.buildNumericField({
  fieldKey: "maxVal",
  fieldProps: { label: "Maximum Value" },
  inputProps: {},
});

export interface MinMaxValueFieldsProps {
  path: string;
}

export const MinMaxValueFields = ({ path }: MinMaxValueFieldsProps) => (
  <Flex.Box x>
    <MinValueField path={path} grow />
    <MaxValueField path={path} grow />
  </Flex.Box>
);
