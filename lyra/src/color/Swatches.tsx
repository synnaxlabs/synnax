// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/color/Swatches.css";

import { color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Button } from "@/button";
import { BaseSwatch } from "@/color/BaseSwatch";
import { useFrequent } from "@/color/Provider";
import { CSS } from "@/css";
import { Divider } from "@/divider";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Theming } from "@/theming";

export interface SwatchesProps {
  /** The picked color. The swatch that matches it shows as selected. */
  value?: color.Crude;
  /** Called with the picked color, or with undefined when the user picks Auto. */
  onChange: (value?: color.Color) => void;
  /**
   * The color the theme paints while the value is absent. Setting it adds an Auto
   * button that clears the value.
   */
  fallback?: color.Crude;
}

/**
 * One-click swatches for the theme colors and the visualization palette, then the
 * user's frequent colors from the nearest {@link Provider}.
 */
export const Swatches = ({
  value,
  onChange,
  fallback,
}: SwatchesProps): ReactElement => {
  const { colors } = Theming.use();
  const frequent = useFrequent();
  const presets = [
    colors.primary.z,
    colors.secondary.z,
    colors.warning.z,
    colors.error.z,
    ...colors.visualization.palettes.default,
  ];
  return (
    <Flex.Box y gap="medium" className={CSS.B("color-swatches")}>
      <div className={CSS.BE("color-swatches", "grid")}>
        {fallback != null && (
          <Button.Button
            variant="outlined"
            size="tiny"
            className={CSS.cls(
              CSS.BE("color-swatches", "auto"),
              CSS.selected(value == null),
            )}
            onClick={() => onChange(undefined)}
            tooltip="Auto: the theme picks the color"
            aria-label="Auto"
          >
            <Icon.Auto />
            Auto
          </Button.Button>
        )}
        {presets.map((c) => (
          <PickerSwatch
            key={color.hex(c)}
            value={c}
            current={value}
            onChange={onChange}
          />
        ))}
      </div>
      {frequent.length > 0 && (
        <>
          <Divider.Divider x />
          <div className={CSS.BE("color-swatches", "grid")}>
            {frequent.map((c) => (
              <PickerSwatch
                key={color.hex(c)}
                value={c}
                current={value}
                onChange={onChange}
              />
            ))}
          </div>
        </>
      )}
    </Flex.Box>
  );
};

interface PickerSwatchProps {
  value: color.Crude;
  /** The picked color. The swatch shows as selected when it matches. */
  current?: color.Crude;
  onChange: (value: color.Color) => void;
}

const PickerSwatch = ({
  value,
  current,
  onChange,
}: PickerSwatchProps): ReactElement => (
  <BaseSwatch
    value={value}
    className={CSS.cls(CSS.selected(current != null && color.equals(current, value)))}
    size="tiny"
    onClick={() => onChange(color.construct(value))}
    aria-label={color.hex(value)}
  />
);
