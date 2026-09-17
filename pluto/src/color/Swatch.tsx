// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/color/Swatch.css";

import { type color, state as xstate } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo, useState } from "react";

import { BaseSwatch, type BaseSwatchProps } from "@/color/BaseSwatch";
import { Picker, type PickerProps } from "@/color/Picker";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { state } from "@/state";
import { Text } from "@/text";

export interface SwatchProps
  extends
    BaseSwatchProps,
    Pick<Dialog.FrameProps, "visible" | "onVisibleChange" | "initialVisible">,
    Pick<PickerProps, "onDelete" | "position"> {
  allowChange?: boolean;
  onlyChangeOnBlur?: boolean;
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
  onChange,
  onVisibleChange,
  initialVisible = false,
  allowChange = true,
  onlyChangeOnBlur = false,
  style,
  onClick,
  value,
  visible: propsVisible,
  ...rest
}: SwatchProps): ReactElement => {
  const [visible, setVisible] = state.usePassthrough({
    initial: initialVisible,
    value: propsVisible,
    onChange: onVisibleChange,
  });
  const [pending, setPending] = useState<color.Color | null>(null);
  const handleVisibleChange = useCallback<xstate.Setter<boolean>>(
    (arg) => {
      if (pending != null && !xstate.executeSetter(arg, visible)) {
        onChange?.(pending);
        setPending(null);
      }
      setVisible(arg);
    },
    [visible, pending, onChange, setVisible],
  );
  const handleSwatchChange = useCallback<NonNullable<BaseSwatchProps["onChange"]>>(
    (c) => {
      setPending(null);
      onChange?.(c);
    },
    [onChange],
  );
  const handlePickerChange = useCallback<PickerProps["onChange"]>(
    (c) => {
      if (onlyChangeOnBlur) setPending(c);
      else onChange?.(c);
    },
    [onlyChangeOnBlur, onChange],
  );
  const canPick = onChange != null && allowChange;
  const handleClick = useCallback<NonNullable<BaseSwatchProps["onClick"]>>(
    (e) => (canPick ? handleVisibleChange(true) : onClick?.(e)),
    [canPick, handleVisibleChange, onClick],
  );
  const tooltip = useMemo(
    () =>
      canPick ? <Text.Text level="small">Click to change color</Text.Text> : undefined,
    [canPick],
  );
  const shownValue = pending ?? value;
  const swatch = (
    <BaseSwatch
      disabled={!canPick && onClick == null}
      onClick={handleClick}
      onChange={handleSwatchChange}
      value={shownValue}
      style={style}
      tooltip={tooltip}
      {...rest}
    />
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
        <Picker value={shownValue} onChange={handlePickerChange} />
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
