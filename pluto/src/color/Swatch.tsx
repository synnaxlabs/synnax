// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/color/Swatch.css";

import { type ReactElement, useCallback, useMemo } from "react";

import { BaseSwatch, type BaseSwatchProps } from "@/color/BaseSwatch";
import { Picker, type PickerProps } from "@/color/Picker";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { state } from "@/state";
import { Text } from "@/text";

export interface SwatchProps
  extends BaseSwatchProps,
    Pick<Dialog.FrameProps, "visible" | "onVisibleChange" | "initialVisible">,
    Pick<PickerProps, "onDelete" | "position"> {
  allowChange?: boolean;
}

export const Swatch = ({
  onChange,
  onVisibleChange,
  initialVisible = false,
  allowChange = true,
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
  const canPick = onChange != null && allowChange;
  const handleClick = useCallback<NonNullable<BaseSwatchProps["onClick"]>>(
    (e) => (canPick ? setVisible(true) : onClick?.(e)),
    [canPick, setVisible, onClick],
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
      onChange={onChange}
      value={value}
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
      onVisibleChange={setVisible}
      className={CSS.BE("color-swatch", "dropdown")}
      variant="floating"
    >
      {swatch}
      <Dialog.Dialog rounded={1}>
        <Picker value={value} onChange={onChange} />
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
