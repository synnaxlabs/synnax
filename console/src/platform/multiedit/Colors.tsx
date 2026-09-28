// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Color } from "@synnaxlabs/lyra/color";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Input } from "@synnaxlabs/lyra/input";
import { color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { type ColorRef, groupByColor } from "@/platform/multiedit/config";

export interface ColorFieldProps {
  label: string;
  /** The color of each element that has the field. Undefined means absent. */
  values: Array<color.Crude | undefined>;
  /** The color the theme paints while the field is absent. */
  fallback: color.Crude;
  /** Called with the new color, or with undefined when the user picks Auto. */
  onChange: (value?: color.Color) => void;
}

/**
 * Edits one color field across several elements. It shows the shared color, Auto when
 * every element leaves the field absent, and Mixed when the elements differ.
 */
export const ColorField = ({
  label,
  values,
  fallback,
  onChange,
}: ColorFieldProps): ReactElement => {
  const [first] = values;
  const mixed = values.some((v) => !color.equals(v, first));
  return (
    <Input.Item label={label} align="start" padHelpText={false}>
      <Color.Input
        aria-label={label}
        value={mixed ? undefined : first}
        mixed={mixed}
        fallback={fallback}
        onChange={onChange}
      />
    </Input.Item>
  );
};

export interface SelectionColorsProps {
  refs: ColorRef[];
  /** Called with every reference that held the changed color. */
  onChange: (refs: ColorRef[], value: color.Color) => void;
}

/** A swatch for each color the selection stores. A change recolors every holder. */
export const SelectionColors = ({
  refs,
  onChange,
}: SelectionColorsProps): ReactElement | null => {
  if (refs.length === 0) return null;
  return (
    <Input.Item label="Selection colors" align="start" padHelpText={false}>
      <Flex.Box x wrap>
        {Array.from(groupByColor(refs).entries()).map(([hex, group]) => (
          <Color.Swatch
            key={`${group[0].key}.${group[0].path}`}
            value={hex}
            onChange={(c: color.Color) => onChange(group, c)}
            onlyChangeOnBlur
          />
        ))}
      </Flex.Box>
    </Input.Item>
  );
};
