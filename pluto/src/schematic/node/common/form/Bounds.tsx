// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { bounds } from "@synnaxlabs/x";
import { type ReactElement } from "react";

const STEP = 10;

export interface BoundsFieldsProps extends Pick<Input.ItemProps, "padHelpText"> {
  /** Path to the bounds within the symbol's config. */
  path: string;
  /** The range the symbol maps while the field is absent. */
  fallback: bounds.Bounds;
}

/**
 * The minimum and maximum of a range, each capped by the other so the pair cannot
 * cross. Editing either one stores the pair.
 */
export const BoundsFields = ({
  path,
  fallback,
  padHelpText,
}: BoundsFieldsProps): ReactElement | null => {
  const field = Form.useField<bounds.Bounds | undefined>(path, { optional: true });
  if (field == null) return null;
  const value = field.value ?? fallback;
  return (
    <>
      <Input.Item label="Min value" padHelpText={padHelpText}>
        <Input.Numeric
          value={value.lower}
          step={STEP}
          bounds={{ ...bounds.INFINITE, upper: value.upper }}
          onChange={(lower) => field.onChange({ ...value, lower })}
        />
      </Input.Item>
      <Input.Item label="Max value" padHelpText={padHelpText}>
        <Input.Numeric
          value={value.upper}
          step={STEP}
          bounds={{ ...bounds.INFINITE, lower: value.lower }}
          onChange={(upper) => field.onChange({ ...value, upper })}
        />
      </Input.Item>
    </>
  );
};
