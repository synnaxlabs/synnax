// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type color } from "@synnaxlabs/client";
import { Color as LyraColor } from "@synnaxlabs/lyra/color";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Input } from "@synnaxlabs/lyra/input";
import { Color } from "@synnaxlabs/pluto";
import { color as xcolor, state } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { type ColorRef } from "@/platform/multiedit/config";
import { type ColorValue } from "@/platform/multiedit/selection";

const refID = (ref?: color.Ref): string => {
  if (ref == null) return "";
  return ref.type === "named" ? ref.key : xcolor.hex(ref.value);
};

export interface ColorFieldProps {
  label: string;
  /** The color of each element that has the field. Must not be empty. */
  values: ColorValue[];
  /**
   * Called with a named reference for a pick from a palette, a fixed reference for a
   * typed or custom color, and undefined when the user picks Auto.
   */
  onChange: (value?: color.Ref) => void;
  /** If true, the picker hides the palettes, for a field that stores fixed colors. */
  palettesHidden?: boolean;
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
  palettesHidden = false,
}: ColorFieldProps): ReactElement => {
  const resolve = Color.useResolve();
  const [first] = values;
  const painted = (v: ColorValue): xcolor.Crude => resolve(v.value) ?? v.fallback;
  const mixed = values.some(
    (v) =>
      refID(v.value) !== refID(first.value) ||
      !xcolor.equals(painted(v), painted(first)),
  );
  const binding = Color.useBinding(mixed ? undefined : first.value, onChange);
  return (
    <Input.Item label={label} align="start" padHelpText={false}>
      <LyraColor.Input
        aria-label={label}
        value={binding.value}
        mixed={mixed}
        fallback={first.fallback}
        onChange={binding.onChange}
        library={palettesHidden ? undefined : binding.library}
      />
    </Input.Item>
  );
};

export interface SelectionColorsProps {
  refs: ColorRef[];
  /** Called with every reference that held the changed color. */
  onChange: (refs: ColorRef[], value: color.Ref) => void;
}

const holderID = (ref: ColorRef): string => `${ref.key}.${ref.path}`;

interface Held {
  id: string;
  groups: ColorRef[][];
}

/**
 * A swatch for each color the selection stores. A change recolors every holder. A
 * swatch hides the palettes while a holder is value only.
 */
export const SelectionColors = ({
  refs,
  onChange,
}: SelectionColorsProps): ReactElement | null => {
  // An open picker holds the grouping, so dragging through another group's color does
  // not merge the two groups.
  const [held, setHeld] = useState<Held | null>(null);
  if (refs.length === 0) return null;
  const live = new Map(refs.map((r) => [holderID(r), r]));
  const groups =
    held?.groups ?? Array.from(Map.groupBy(refs, (r) => refID(r.value)).values());
  return (
    <Input.Item label="Selection" align="start" padHelpText={false}>
      <Flex.Box x wrap>
        {groups.map((group) => {
          const current = group.flatMap((r) => live.get(holderID(r)) ?? []);
          if (current.length === 0) return null;
          const id = holderID(group[0]);
          const visible = held?.id === id;
          return (
            <GroupSwatch
              key={id}
              value={current[0].value}
              visible={visible}
              onVisibleChange={(arg) =>
                setHeld(state.executeSetter(arg, visible) ? { id, groups } : null)
              }
              onChange={(c) => onChange(current, c)}
              palettesHidden={current.some((r) => r.valueOnly === true)}
            />
          );
        })}
      </Flex.Box>
    </Input.Item>
  );
};

interface GroupSwatchProps extends Pick<
  LyraColor.SwatchProps,
  "visible" | "onVisibleChange"
> {
  value: color.Ref;
  onChange: (value: color.Ref) => void;
  palettesHidden: boolean;
}

const GroupSwatch = ({
  value,
  onChange,
  palettesHidden,
  ...rest
}: GroupSwatchProps): ReactElement | null => {
  const binding = Color.useBinding(value, (c) => {
    if (c != null) onChange(c);
  });
  // A named color shows once the named colors load.
  if (binding.value == null) return null;
  return (
    <LyraColor.Swatch
      {...rest}
      value={binding.value}
      onChange={binding.onChange}
      library={palettesHidden ? undefined : binding.library}
    />
  );
};
