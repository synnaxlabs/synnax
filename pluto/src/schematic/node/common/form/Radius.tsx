// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type border, xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Form as Base } from "@/form";
import { Input } from "@/input";
import { PERCENT_BORDER_RADIUS_INPUT_PROPS } from "@/schematic/node/common/form/input";

export interface RadiusFieldsProps {
  path: string;
}

/**
 * RadiusFields edits a symbol's corner radius through one x and one y percentage,
 * writing the pair into every corner. The stored shape carries corners because the
 * renderer clips against them; no symbol has ever varied the radius by corner.
 */
export const RadiusFields = ({ path }: RadiusFieldsProps): ReactElement | null => {
  const field = Base.useField<border.Radius | undefined>(path, { optional: true });
  if (field == null) return null;
  const corner = field.value?.topLeft ?? xy.ZERO;
  const set = (next: xy.XY): void =>
    field.onChange({
      topLeft: next,
      topRight: next,
      bottomLeft: next,
      bottomRight: next,
    });
  return (
    <>
      <Input.Item label="X border radius" grow padHelpText={false}>
        <Input.Numeric
          value={corner.x}
          onChange={(x) => set({ ...corner, x })}
          {...PERCENT_BORDER_RADIUS_INPUT_PROPS}
        />
      </Input.Item>
      <Input.Item label="Y border radius" grow padHelpText={false}>
        <Input.Numeric
          value={corner.y}
          onChange={(y) => set({ ...corner, y })}
          {...PERCENT_BORDER_RADIUS_INPUT_PROPS}
        />
      </Input.Item>
    </>
  );
};
