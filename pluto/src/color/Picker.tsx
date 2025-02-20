// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/color/Picker.css";

import { Icon } from "@synnaxlabs/media";
import { type ComponentPropsWithoutRef, useCallback } from "react";
import { type ColorResult, SketchPicker } from "react-color";

import { Align } from "@/align";
import { Button } from "@/button";
import { BaseSwatch } from "@/color/BaseSwatch";
import { color } from "@/color/core";
import { useFrequent, useFrequentUpdater } from "@/color/Provider";
import { CSS } from "@/css";
import { useDebouncedCallback } from "@/hooks";
import { type Input } from "@/input";
import { Text } from "@/text";

export interface PickerProps
  extends Input.Control<color.Crude, color.Color>,
    Omit<ComponentPropsWithoutRef<"div">, "onChange"> {
  onDelete?: () => void;
  position?: number;
}

export const Picker = ({
  value,
  onChange,
  position,
  onDelete,
  ...rest
}: PickerProps) => {
  const updateFreq = useFrequentUpdater();
  const updateFreqDebounced = useDebouncedCallback(updateFreq, 1000, [updateFreq]);

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
      const c = new color.Color(res.hex, res.rgb.a);
      baseHandleChange(c);
    },
    [baseHandleChange, updateFreqDebounced],
  );

  return (
    <Align.Space
      direction="y"
      align="start"
      className={CSS.B("color-picker-container")}
    >
      {position != null ||
        (onDelete != null && (
          <Align.Space direction="x" justify="spaceBetween">
            {position != null && (
              <Text.Text level="small" shade={7}>
                {position} %
              </Text.Text>
            )}
            {onDelete != null && (
              <Button.Icon name="close" onClick={onDelete} size="small">
                <Icon.Delete />
              </Button.Icon>
            )}
          </Align.Space>
        ))}
      <SketchPicker
        className={CSS.B("color-picker")}
        color={new color.Color(value).hex}
        onChange={pickerHandleChange}
        presetColors={[]}
        {...rest}
      />
      <Frequent onChange={baseHandleChange} />
    </Align.Space>
  );
};

interface FrequentProps extends Omit<ComponentPropsWithoutRef<"div">, "onChange"> {
  onChange?: (value: color.Color) => void;
}

const Frequent = ({ onChange }: FrequentProps) => {
  const frequent = useFrequent();
  return (
    <Align.Space direction="x" wrap size={0.5}>
      {frequent.map((c, i) => (
        <BaseSwatch key={i} value={c} size="tiny" onClick={() => onChange?.(c)} />
      ))}
    </Align.Space>
  );
};
