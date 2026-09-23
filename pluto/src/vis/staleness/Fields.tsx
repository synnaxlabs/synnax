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
import { Form } from "@/form";
import { Theming } from "@/theming";
import { staleness } from "@/vis/staleness/aether";

export interface FieldsProps {
  /** Path to the config holding the staleness keys. Defaults to the form root. */
  path?: string;
}

/**
 * Fields edits the color a component takes on, and the delay before it does, once its
 * source stops sending. It renders as a pair of siblings, so the caller places it in a
 * row of its own choosing. An unchosen color is absent, so the swatch shows the theme
 * color it resolves to until a pick writes one.
 */
export const Fields = ({ path = "" }: FieldsProps = {}): ReactElement => {
  const theme = Theming.use();
  const field = (name: string): string =>
    path.length === 0 ? name : `${path}.${name}`;
  return (
    <>
      <Form.Field<color.Crude>
        label="Color"
        align="start"
        padHelpText={false}
        path={field("stalenessColor")}
        defaultValue={staleness.resolveColor(undefined, theme)}
      >
        {({ value, onChange }) => (
          <Color.Swatch value={value} onChange={onChange} bordered />
        )}
      </Form.Field>
      <Form.NumericField
        path={field("stalenessTimeout")}
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
