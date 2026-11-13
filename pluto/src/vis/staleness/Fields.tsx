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
import { Input } from "@synnaxlabs/lyra/input";
import { Theming } from "@synnaxlabs/lyra/theming";
import { type color, deep, TimeSpan } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { staleness } from "@/vis/staleness/aether";

const path = deep.path<{ stalenessColor?: color.Color; stalenessTimeout?: number }>();

export interface FieldsProps {
  /** The unit of the stored timeout. Defaults to nanoseconds. */
  unit?: TimeSpan;
}

// The Core stores the timeout as int64 nanoseconds, and a client holds it as a float64,
// so the Core and the client bound it at 2^53 - 1 ns.
const MAX_TIMEOUT = new TimeSpan(Number.MAX_SAFE_INTEGER);

/** @returns why a staleness timeout is out of bounds, or null when it is valid. */
export const checkTimeout = (span: TimeSpan): string | null => {
  if (span.lessThan(TimeSpan.SECOND)) return "Timeout must be 1s or more";
  if (span.greaterThan(MAX_TIMEOUT))
    return `Timeout must be ${MAX_TIMEOUT.toString()} or less`;
  return null;
};

/**
 * Fields edits the color a component takes on, and the delay before it does, once its
 * source stops sending. It renders as a pair of siblings, so the caller places it in a
 * row of its own choosing. An unchosen color is absent, so the swatch shows the theme
 * color it resolves to until a pick writes one. A number typed alone is seconds.
 */
export const Fields = ({ unit = TimeSpan.NANOSECOND }: FieldsProps): ReactElement => {
  const theme = Theming.use();
  return (
    <>
      <Color.Field
        path={path.stalenessColor}
        fallback={staleness.resolveColor(undefined, theme)}
      />
      <Form.Field<number | undefined>
        path={path.stalenessTimeout}
        label="Timeout"
        padHelpText={false}
      >
        {({ value, onChange, ...p }) => (
          <Input.TimeSpan
            {...p}
            {...INPUT_PROPS}
            inline
            value={value == null ? undefined : Number(unit.mult(value).valueOf())}
            onChange={(ns) => onChange(ns / Number(unit.valueOf()))}
            check={checkTimeout}
          />
        )}
      </Form.Field>
    </>
  );
};

const INPUT_PROPS = { style: { maxWidth: "18rem" } } as const;
