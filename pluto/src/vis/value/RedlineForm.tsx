// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type bounds, color, deep, scale } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Color } from "@/color";
import { Flex } from "@/flex";
import { Form } from "@/form";
import { type Redline, ZERO_READLINE } from "@/vis/value/redline";

const boundsInputProps = { size: "small", showDragHandle: false } as const;

const boundsStyle = { width: 60 };

export interface RedlineFormProps {
  path: string;
}

const baseScale = scale.Scale.scale<number>(0, 1);

export const RedlineForm = ({ path }: RedlineFormProps): ReactElement => {
  const { set, get } = Form.useContext();
  const { bounds } = Form.useFieldValue<Redline>(`${path}`, {
    defaultValue: {
      bounds: { ...ZERO_READLINE.bounds },
      gradient: [...ZERO_READLINE.gradient],
    },
  });
  const scale = baseScale.scale(bounds);
  return (
    <Flex.Box x grow>
      <Form.NumericField
        inputProps={boundsInputProps}
        style={boundsStyle}
        label="Lower"
        path={`${path}.bounds.lower`}
      />
      <Form.Field<color.Gradient>
        path={`${path}.gradient`}
        label="Gradient"
        align="start"
        padHelpText={false}
      >
        {({ value, onChange }) => (
          <Color.GradientPicker
            value={deep.copy(value)}
            scale={scale}
            onChange={(v) => {
              const prevB = get<bounds.Bounds>(`${path}.bounds`).value;
              const nextBounds = { ...prevB };
              const positions = v.map((c) => c.position);
              const highestPos = scale.pos(Math.max(...positions));
              const lowestPos = scale.pos(Math.min(...positions));
              const highestGreater = highestPos > nextBounds.upper;
              const lowestLower = lowestPos < nextBounds.lower;
              if (highestGreater) {
                v[v.length - 1].position = 1;
                nextBounds.upper = highestPos;
              }
              if (lowestLower) {
                v[0].position = 0;
                nextBounds.lower = lowestPos;
              }
              const nextGradient = v.map((c) => ({ ...c, color: color.hex(c.color) }));
              if (highestGreater || lowestLower)
                set(path, { bounds: nextBounds, gradient: nextGradient });
              else onChange(v.map((c) => ({ ...c, color: color.hex(c.color) })));
            }}
          />
        )}
      </Form.Field>
      <Form.NumericField
        inputProps={boundsInputProps}
        style={boundsStyle}
        label="Upper"
        path={`${path}.bounds.upper`}
      />
    </Flex.Box>
  );
};
