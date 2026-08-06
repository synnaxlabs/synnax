// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Color } from "@/color";
import { Flex } from "@/flex";
import { Form } from "@/form";
import { Theming } from "@/theming";
import { staleness } from "@/vis/staleness/aether";

/// Fields edits the color a component takes on, and the delay before it does, once its
/// source stops sending.
export const Fields = (): ReactElement => {
  const theme = Theming.use();
  return (
    <Flex.Box x>
      <Form.Field<color.Crude>
        hideIfNull
        label="Stale color"
        align="start"
        padHelpText={false}
        path="stalenessColor"
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
        inputProps={{ bounds: { lower: 1, upper: Infinity }, endContent: "s" }}
      />
    </Flex.Box>
  );
};
