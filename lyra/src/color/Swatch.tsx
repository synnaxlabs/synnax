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
import { type ReactElement, useCallback, useMemo, useState } from "react";

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
  allowChange?: boolean;
  onlyChangeOnBlur?: boolean;
}

interface RequiredValueProps extends BaseProps {
  value: color.Crude;
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

/**
 * A color swatch that opens a picker when clicked.
 * @param props - The props for the swatch. Unlisted props are passed to the underlying
 * button.
 * @param props.onChange - A function to call when the color changes.
 * @param props.onlyChangeOnBlur - If true, the swatch holds picker changes locally and
 * calls `onChange` once, when the picker closes. Set it where a live preview is not
 * worth a change per pixel of drag through the gradient. A change pending when the
 * swatch unmounts is dropped.
 */
export const Swatch = ({
  onChange: propsOnChange,
  onVisibleChange,
  initialVisible = false,
  allowChange = true,
  onlyChangeOnBlur = false,
  style,
  onClick,
  value,
  fallback,
  visible: propsVisible,
  className,
  ...rest
}: SwatchProps): ReactElement => {
  // Only an OptionalValueProps caller sets a fallback, and only the fallback's Auto
  // swatch passes undefined.
  const onChange = propsOnChange as ((value?: color.Color) => void) | undefined;
  const [visible, setVisible] = state.usePassthrough({
    initial: initialVisible,
    value: propsVisible,
    onChange: onVisibleChange,
  });
  const [pending, setPending] = useState<Pending | null>(null);
  const handleVisibleChange = useCallback<xstate.Setter<boolean>>(
    (arg) => {
      if (pending != null && !xstate.executeSetter(arg, visible)) {
        onChange?.(pending.value);
        setPending(null);
      }
      setVisible(arg);
    },
    [visible, pending, onChange, setVisible],
  );
  const handleSwatchChange = useCallback(
    (c: color.Color) => {
      setPending(null);
      onChange?.(c);
    },
    [onChange],
  );
  const handlePickerChange = useCallback(
    (c?: color.Color) => {
      if (onlyChangeOnBlur) setPending({ value: c });
      else onChange?.(c);
    },
    [onlyChangeOnBlur, onChange],
  );
  const canPick = onChange != null && allowChange;
  const handleClick = useCallback<NonNullable<BaseSwatchProps["onClick"]>>(
    (e) => (canPick ? handleVisibleChange(true) : onClick?.(e)),
    [canPick, handleVisibleChange, onClick],
  );
  const shownValue = pending != null ? pending.value : value;
  const auto = shownValue == null;
  const tooltip = useMemo(() => {
    if (!canPick) return undefined;
    return (
      <Text.Text level="small">
        {auto
          ? "Auto: the theme picks the color. Click to change."
          : "Click to change color"}
      </Text.Text>
    );
  }, [canPick, auto]);
  const swatch = (
    <BaseSwatch
      disabled={!canPick && onClick == null}
      onClick={handleClick}
      onChange={handleSwatchChange}
      value={shownValue ?? fallback ?? color.ZERO}
      style={style}
      tooltip={tooltip}
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
  if (!canPick) return swatch;
  return (
    <Dialog.Frame
      visible={visible}
      initialVisible={initialVisible}
      onVisibleChange={handleVisibleChange}
      className={CSS.BE("color-swatch", "dropdown")}
      variant="floating"
    >
      {swatch}
      <Dialog.Dialog rounded="small">
        <Picker value={shownValue} fallback={fallback} onChange={handlePickerChange} />
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
