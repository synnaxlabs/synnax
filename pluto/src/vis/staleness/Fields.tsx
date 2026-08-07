// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Color } from "@/color";
import { Flex } from "@/flex";
import { Form } from "@/form";
import { Theming } from "@/theming";
import { staleness } from "@/vis/staleness/aether";

/// Fields edits the color a component takes on, and the delay before it does, once its
/// source stops sending. Both fields carry a default so they still render on a symbol
/// saved before it gained staleness config.
export const Fields = (): ReactElement => {
  const theme = Theming.use();
  return (
    <Flex.Box x>
      <Form.Field<color.Crude>
        label="Stale color"
        align="start"
        padHelpText={false}
        path="stalenessColor"
        defaultValue={color.ZERO}
      >
        {({ value, onChange }) => (
          <Color.Swatch
            value={staleness.resolveColor(value, theme)}
            onChange={onChange}
            bordered
          />
        )}
      </Form.Field>
      <Form.NumericField
        path="stalenessTimeout"
        label="Stale timeout"
        padHelpText={false}
        defaultValue={staleness.DEFAULT_TIMEOUT}
        inputProps={INPUT_PROPS}
      />
    </Flex.Box>
  );
};

const INPUT_PROPS: Form.NumericFieldProps["inputProps"] = {
  bounds: { lower: 1, upper: Infinity },
  endContent: "s",
};
