// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/color/Swatch.css";

import { color, state as xstate } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { BaseSwatch, type BaseSwatchProps } from "@/color/BaseSwatch";
import { Picker } from "@/color/Picker";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Icon } from "@/icon";
import { state } from "@/state";
import { Text } from "@/text";

interface BaseProps
  extends
    Omit<BaseSwatchProps, "value" | "onChange">,
    Pick<Dialog.FrameProps, "visible" | "onVisibleChange" | "initialVisible"> {
  /**
   * If true, the swatch holds picker changes and calls `onChange` once, when the picker
   * closes. A change pending when the swatch unmounts is dropped.
   */
  onlyChangeOnBlur?: boolean;
}

interface RequiredValueProps extends BaseProps {
  value: color.Crude;
  /** Without it, the swatch only shows the color. */
  onChange?: (value: color.Color) => void;
  fallback?: undefined;
}

interface OptionalValueProps extends BaseProps {
  value?: color.Crude;
  /** Called with undefined when the user picks Auto. */
  onChange?: (value?: color.Color) => void;
  /**
   * The color the theme paints while the value is absent. Setting it makes the value
   * optional: the picker offers an Auto button that clears it, and the swatch marks
   * when it shows the fallback.
   */
  fallback: color.Crude;
}

export type SwatchProps = RequiredValueProps | OptionalValueProps;

/** A change the swatch holds until the picker closes. */
interface Pending {
  value?: color.Color;
}

/** A color swatch that opens a picker when clicked. Takes a dropped color too. */
export const Swatch = ({
  onChange: propsOnChange,
  onVisibleChange,
  initialVisible = false,
  onlyChangeOnBlur = false,
  onClick,
  value,
  fallback,
  visible: propsVisible,
  className,
  ...rest
}: SwatchProps): ReactElement => {
  // Only an OptionalValueProps caller sets a fallback, and only then does the picker
  // show the Auto button that passes undefined.
  const onChange = propsOnChange as ((value?: color.Color) => void) | undefined;
  const [visible, setVisible] = state.usePassthrough({
    initial: initialVisible,
    value: propsVisible,
    onChange: onVisibleChange,
  });
  const [pending, setPending] = useState<Pending | null>(null);
  const handleVisibleChange: xstate.Setter<boolean> = (arg) => {
    if (pending != null && !xstate.executeSetter(arg, visible)) {
      onChange?.(pending.value);
      setPending(null);
    }
    setVisible(arg);
  };
  const shownValue = pending != null ? pending.value : value;
  const auto = shownValue == null;
  const swatch = (
    <BaseSwatch
      disabled={onChange == null && onClick == null}
      onClick={(e) => (onChange != null ? handleVisibleChange(true) : onClick?.(e))}
      onChange={
        onChange == null
          ? undefined
          : (c) => {
              setPending(null);
              onChange(c);
            }
      }
      value={shownValue ?? fallback ?? color.ZERO}
      tooltip={
        onChange == null ? undefined : (
          <Text.Text level="small">
            {auto
              ? "Auto: the theme picks the color. Click to change."
              : "Click to change color"}
          </Text.Text>
        )
      }
      className={CSS.cls(
        CSS.BM("color-swatch", "chip"),
        auto && CSS.M("auto"),
        className,
      )}
      {...rest}
    >
      {auto && <Icon.Auto />}
    </BaseSwatch>
  );
  if (onChange == null) return swatch;
  return (
    <Dialog.Frame
      visible={visible}
      onVisibleChange={handleVisibleChange}
      className={CSS.BE("color-swatch", "dropdown")}
      variant="floating"
    >
      {swatch}
      <Dialog.Dialog>
        <Picker
          value={shownValue}
          fallback={fallback}
          onChange={(c) => (onlyChangeOnBlur ? setPending({ value: c }) : onChange(c))}
        />
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
