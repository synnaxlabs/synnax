// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/color/Picker.css";

import { color } from "@synnaxlabs/x";
import { type ComponentPropsWithoutRef, type ReactElement, useCallback } from "react";
import { type ColorResult, SketchPicker } from "react-color";

import { Button } from "@/button";
import { BaseSwatch } from "@/color/BaseSwatch";
import { useFrequent, useFrequentUpdater } from "@/color/Provider";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { useDebouncedCallback } from "@/hooks";
import { Icon } from "@/icon";
import { type Input } from "@/input";
import { Text } from "@/text";

export interface PickerProps
  extends
    Input.Control<color.Crude, color.Color>,
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
}: PickerProps): ReactElement => {
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
      const c = color.construct(res.hex, res.rgb.a);
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
      <SketchPicker
        className={CSS.B("color-picker")}
        color={color.hex(value)}
        onChange={pickerHandleChange}
        presetColors={[]}
        {...rest}
      />
      <Frequent onChange={baseHandleChange} />
    </Flex.Box>
  );
};

interface FrequentProps extends Omit<ComponentPropsWithoutRef<"div">, "onChange"> {
  onChange?: (value: color.Color) => void;
}

const Frequent = ({ onChange }: FrequentProps) => {
  const frequent = useFrequent();
  return (
    <Flex.Box x wrap gap="tiny">
      {frequent.map((c, i) => (
        <BaseSwatch key={i} value={c} size="tiny" onClick={() => onChange?.(c)} />
      ))}
    </Flex.Box>
  );
};
