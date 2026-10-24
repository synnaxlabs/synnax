// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Color } from "@synnaxlabs/lyra/color";
import { Form } from "@synnaxlabs/lyra/form";
import { Theming } from "@synnaxlabs/lyra/theming";
import { type ReactElement } from "react";

import { staleness } from "@/vis/staleness/aether";

/**
 * Fields edits the color a component takes on, and the delay before it does, once its
 * source stops sending. It renders as a pair of siblings, so the caller places it in a
 * row of its own choosing. An unchosen color is absent, so the swatch shows the theme
 * color it resolves to until a pick writes one.
 */
export const Fields = (): ReactElement => {
  const theme = Theming.use();
  return (
    <>
      <Color.Field
        path="stalenessColor"
        fallback={staleness.resolveColor(undefined, theme)}
      />
      <Form.NumericField
        path="stalenessTimeout"
        label="Timeout"
        padHelpText={false}
        inputProps={INPUT_PROPS}
      />
    </>
  );
};

const INPUT_PROPS: Form.NumericFieldProps["inputProps"] = {
  bounds: { lower: 1, upper: Infinity },
  endContent: "s",
  style: { maxWidth: "18rem" },
};
