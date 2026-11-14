// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Color } from "@synnaxlabs/lyra/color";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { type Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Theming } from "@synnaxlabs/lyra/theming";
import { type color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

/** Props for {@link Picker}. */
export interface PickerProps {
  icon?: string;
  color?: color.Crude;
  /** The icon drawn while the icon is absent. */
  fallback: Icon.FC;
  onIconChange: (next?: string) => void;
  onColorChange: (next?: color.Color) => void;
}

/**
 * Picks an icon and its color, in a {@link Dialog}. A color pick keeps the dialog open,
 * so the user can see the color on each icon. An icon pick closes it.
 */
export const Picker = ({
  icon,
  color,
  fallback,
  onIconChange,
  onColorChange,
}: PickerProps): ReactElement => {
  const { colors } = Theming.use();
  const { close } = Dialog.useContext();
  const updateFrequent = Color.useFrequentUpdater();
  // The custom picker counts its own picks as frequent.
  const handlePresetChange = (next?: color.Color) => {
    onColorChange(next);
    if (next != null) updateFrequent(next);
  };
  const handleIconChange = (next?: string) => {
    onIconChange(next);
    close();
  };
  return (
    <Flex.Box y gap="large" className={CSS.B("appearance-picker")}>
      <Color.Swatches
        value={color}
        onChange={handlePresetChange}
        fallback={colors.gray.l10}
      >
        <Color.Swatch
          custom
          value={color}
          fallback={colors.gray.l10}
          onChange={onColorChange}
          onlyChangeOnBlur
          size="tiny"
        />
      </Color.Swatches>
      <Input.IconPicker
        value={icon}
        onChange={handleIconChange}
        color={color}
        fallback={fallback}
      />
    </Flex.Box>
  );
};
