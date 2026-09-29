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
import { color, state } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { type ColorRef, groupByColor } from "@/platform/multiedit/config";
import { type ColorValue } from "@/platform/multiedit/selection";

export interface ColorFieldProps {
  label: string;
  /** The color of each element that has the field. Must not be empty. */
  values: ColorValue[];
  /** Called with the new color, or with undefined when the user picks Auto. */
  onChange: (value?: color.Color) => void;
}

/**
 * Edits one color field across several elements. It shows the shared color, Auto when
 * every element leaves the field absent and paints the same fallback, and Mixed when
 * the elements differ.
 */
export const ColorField = ({
  label,
  values,
  onChange,
}: ColorFieldProps): ReactElement => {
  const [first] = values;
  const painted = (v: ColorValue): color.Crude => v.value ?? v.fallback;
  const mixed = values.some(
    (v) =>
      !color.equals(v.value, first.value) || !color.equals(painted(v), painted(first)),
  );
  return (
    <Input.Item label={label} align="start" padHelpText={false}>
      <Color.Input
        aria-label={label}
        value={mixed ? undefined : first.value}
        mixed={mixed}
        fallback={first.fallback}
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

const refID = (ref: ColorRef): string => `${ref.key}.${ref.path}`;

interface Held {
  id: string;
  groups: ColorRef[][];
}

/** A swatch for each color the selection stores. A change recolors every holder. */
export const SelectionColors = ({
  refs,
  onChange,
}: SelectionColorsProps): ReactElement | null => {
  // An open picker holds the grouping, so dragging through another group's color does
  // not merge the two groups.
  const [held, setHeld] = useState<Held | null>(null);
  if (refs.length === 0) return null;
  const live = new Map(refs.map((r) => [refID(r), r]));
  const groups = held?.groups ?? Array.from(groupByColor(refs).values());
  return (
    <Input.Item label="Selection colors" align="start" padHelpText={false}>
      <Flex.Box x wrap>
        {groups.map((group) => {
          const current = group.flatMap((r) => live.get(refID(r)) ?? []);
          if (current.length === 0) return null;
          const id = refID(group[0]);
          const visible = held?.id === id;
          return (
            <Color.Swatch
              key={id}
              value={current[0].value}
              visible={visible}
              onVisibleChange={(arg) =>
                setHeld(state.executeSetter(arg, visible) ? { id, groups } : null)
              }
              onChange={(c: color.Color) => onChange(current, c)}
            />
          );
        })}
      </Flex.Box>
    </Input.Item>
  );
};
