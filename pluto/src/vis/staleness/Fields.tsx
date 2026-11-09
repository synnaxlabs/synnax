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
import { type color, deep, TimeSpan } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { staleness } from "@/vis/staleness/aether";

const path = deep.path<{ stalenessColor?: color.Color; stalenessTimeout?: number }>();

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
        path={path.stalenessColor}
        fallback={staleness.resolveColor(undefined, theme)}
      />
      <Form.TimeSpanField
        path={path.stalenessTimeout}
        label="Timeout"
        unit={TimeSpan.SECOND}
        bounds={BOUNDS}
        padHelpText={false}
        inputProps={INPUT_PROPS}
      />
    </>
  );
};

const BOUNDS = { lower: TimeSpan.SECOND };

const INPUT_PROPS = { style: { maxWidth: "18rem" } } as const;
