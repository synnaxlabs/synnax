// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/color/Picker.css";

import { Button } from "@synnaxlabs/lyra/button";
import { CSS } from "@synnaxlabs/lyra/css";
import { Flex } from "@synnaxlabs/lyra/flex";
import { useDebouncedCallback } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";
import { color, TimeSpan } from "@synnaxlabs/x";
import { type ComponentPropsWithoutRef, type ReactElement, useCallback } from "react";
import { type ColorResult, SketchPicker } from "react-color";

import { BaseSwatch } from "@/color/BaseSwatch";
import { useFrequent, useFrequentUpdater } from "@/color/Provider";

export interface PickerProps
  extends
    Input.Control<color.Crude, color.Color>,
    Omit<ComponentPropsWithoutRef<"div">, "onChange"> {
  onDelete?: () => void;
  position?: number;
  /** Colors offered ahead of the picker, such as the theme's status colors. */
  presets?: color.Crude[];
}

export const Picker = ({
  value,
  onChange,
  position,
  onDelete,
  presets,
  ...rest
}: PickerProps): ReactElement => {
  const frequent = useFrequent();
  const updateFreq = useFrequentUpdater();
  const updateFreqDebounced = useDebouncedCallback(updateFreq, TimeSpan.SECOND, [
    updateFreq,
  ]);

  const baseHandleChange = useCallback(
    (c: color.Color): void => {
      onChange(c);
      updateFreqDebounced(c);
    },
    [onChange, updateFreqDebounced],
  );

  const pickerHandleChange = useCallback(
    (res: ColorResult): void => {
      if (res.hex === "transparent") onChange(color.ZERO);
      const c = color.setAlpha(res.hex, res.rgb.a ?? 1);
      baseHandleChange(c);
    },
    [baseHandleChange, updateFreqDebounced],
  );

  return (
    <Flex.Box
      y
      align="start"
      className={CSS.B("color-picker-container")}
      background={1}
    >
      {position != null ||
        (onDelete != null && (
          <Flex.Box x justify="between">
            {position != null && <Text.Text level="small">{position} %</Text.Text>}
            {onDelete != null && (
              <Button.Button name="close" onClick={onDelete} size="small">
                <Icon.Delete />
              </Button.Button>
            )}
          </Flex.Box>
        ))}
      {presets != null && <Swatches colors={presets} onChange={baseHandleChange} />}
      <SketchPicker
        className={CSS.B("color-picker")}
        color={color.hex(value)}
        onChange={pickerHandleChange}
        presetColors={[]}
        {...rest}
      />
      <Swatches colors={frequent} onChange={baseHandleChange} />
    </Flex.Box>
  );
};

interface SwatchesProps {
  colors: color.Crude[];
  onChange: (value: color.Color) => void;
}

const Swatches = ({ colors, onChange }: SwatchesProps): ReactElement => (
  <Flex.Box x wrap gap="tiny">
    {colors.map((c, i) => (
      <BaseSwatch
        key={i}
        value={c}
        size="tiny"
        onClick={() => onChange(color.construct(c))}
      />
    ))}
  </Flex.Box>
);
