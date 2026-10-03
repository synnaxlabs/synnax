// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { type Input } from "@synnaxlabs/lyra/input";
import { type ReactElement } from "react";

const ACTIVATION_DELAY_INPUT_PROPS: Partial<Input.NumericProps> = {
  endContent: "ms",
  min: 0,
};

export const ActivationDelayField = (
  props: Partial<Form.NumericFieldProps>,
): ReactElement => (
  <Form.NumericField
    label="Delay"
    path="onClickDelay"
    inputProps={ACTIVATION_DELAY_INPUT_PROPS}
    defaultValue={0}
    padHelpText={false}
    {...props}
  />
);
