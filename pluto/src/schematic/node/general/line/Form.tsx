// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form as Base } from "@synnaxlabs/lyra/form";
import { type Input } from "@synnaxlabs/lyra/input";
import { type ReactElement } from "react";

import { Form } from "@/schematic/node/common/form";

const WIDTH_INPUT_PROPS: Partial<Input.NumericProps> = {
  ...Form.STROKE_WIDTH_INPUT_PROPS,
  bounds: { lower: 1, upper: 21 },
};

export const LineForm = (): ReactElement => (
  <Base.Sections x>
    <Base.Section title="Appearance">
      <Form.ColorField path="color" />
      <Base.NumericField
        path="strokeWidth"
        label="Stroke width"
        padHelpText={false}
        inputProps={WIDTH_INPUT_PROPS}
      />
    </Base.Section>
  </Base.Sections>
);
