// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds } from "@synnaxlabs/x/bounds";
import { xy } from "@synnaxlabs/x/xy";
import { Input } from "@synnaxlabs/charon/input";

import { type ReactElement } from "react";

import { Form as Base } from "@synnaxlabs/charon/form";

const SCALE_BOUNDS: bounds.Bounds = { lower: 5, upper: 1000 };
const SCALE_DRAG_SCALE: xy.Crude = { x: 0.75, y: 0.5 };

export const ScaleField: Base.FieldT<number> = (props): ReactElement => (
  <Base.Field hideIfNull label="Scale" align="start" padHelpText={false} {...props}>
    {({ value, onChange }) => (
      <Input.Numeric
        dragScale={SCALE_DRAG_SCALE}
        bounds={SCALE_BOUNDS}
        endContent="%"
        value={Math.round(value * 100)}
        onChange={(v) => onChange(parseFloat((v / 100).toFixed(2)))}
      />
    )}
  </Base.Field>
);
