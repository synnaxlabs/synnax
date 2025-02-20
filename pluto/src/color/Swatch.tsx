// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/color/Swatch.css";

import { useCallback, useMemo } from "react";

import { BaseSwatch, type BaseSwatchProps } from "@/color/BaseSwatch";
import { Picker, type PickerProps } from "@/color/Picker";
import { CSS } from "@/css";
import { Dropdown } from "@/dropdown";
import { type UseProps } from "@/dropdown/Dropdown";
import { Text } from "@/text";

export interface SwatchProps
  extends BaseSwatchProps,
    UseProps,
    Pick<PickerProps, "onDelete" | "position"> {
  allowChange?: boolean;
}

export const Swatch = ({
  onChange,
  onVisibleChange,
  initialVisible,
  allowChange = true,
  style,
  onClick,
  value,
  ...rest
}: SwatchProps) => {
  const { visible, open, close } = Dropdown.use({ onVisibleChange, initialVisible });
  const canPick = onChange != null && allowChange;
  const handleClick = useCallback<NonNullable<BaseSwatchProps["onClick"]>>(
    (e) => (canPick ? open() : onClick?.(e)),
    [canPick, open, onClick],
  );
  const tooltip = useMemo(
    () =>
      canPick ? <Text.Text level="small">Click to change color</Text.Text> : undefined,
    [canPick],
  );
  const swatch = (
    <BaseSwatch
      disabled={!canPick && onClick == null}
      onClick={handleClick}
      value={value}
      style={style}
      tooltip={tooltip}
      {...rest}
    />
  );
  if (!canPick) return swatch;
  return (
    <Dropdown.Dialog
      close={close}
      visible={visible}
      className={CSS.BE("color-swatch", "dropdown")}
      keepMounted={false}
      variant="floating"
      zIndex={100}
      style={style}
    >
      {swatch}
      <Picker value={value} onChange={onChange} />
    </Dropdown.Dialog>
  );
};
