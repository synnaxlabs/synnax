// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/panel/IconPicker.css";

import { panel } from "@synnaxlabs/client";
import { Color } from "@synnaxlabs/lyra/color";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Theming } from "@synnaxlabs/lyra/theming";
import { Errors, Panel } from "@synnaxlabs/pluto";
import { type color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

const frameID = (key: panel.Key): string => CSS.B(`panel-icon-picker-${key}`);

/**
 * Opens the {@link IconPicker} of the panel.
 *
 * @throws {Error} if no icon picker for the panel is mounted.
 */
export const openIconPicker = (key: panel.Key): void => Dialog.open(frameID(key));

interface ContentProps {
  /** The panel whose icon and color the picker sets. */
  panel: panel.Key;
}

// A color pick keeps the picker open so the user can see it on each icon. An icon pick
// closes it.
const Content = ({ panel: key }: ContentProps): ReactElement => {
  Panel.useEnsure({ key });
  const icon = Panel.useIcon({ key });
  const color = Panel.useColor({ key });
  const { colors } = Theming.use();
  const { close } = Dialog.useContext();
  const { dispatch } = Panel.useDispatch();
  const updateFrequent = Color.useFrequentUpdater();
  // The custom picker counts its own picks as frequent.
  const setColor = (next?: color.Color) =>
    dispatch({ key, actions: panel.setColor({ color: next }) });
  const handlePresetChange = (next?: color.Color) => {
    setColor(next);
    if (next != null) updateFrequent(next);
  };
  const handleIconChange = (next?: string) => {
    dispatch({ key, actions: panel.setIcon({ icon: next }) });
    close();
  };
  return (
    <Flex.Box y gap="large" className={CSS.BE("panel-icon-picker", "content")}>
      <Color.Swatches
        value={color}
        onChange={handlePresetChange}
        fallback={colors.gray.l10}
      >
        <Color.Swatch
          custom
          value={color}
          fallback={colors.gray.l10}
          onChange={setColor}
          onlyChangeOnBlur
          size="tiny"
        />
      </Color.Swatches>
      <Input.IconPicker
        value={icon}
        onChange={handleIconChange}
        color={color}
        fallback={Icon.Panel}
      />
    </Flex.Box>
  );
};

export interface IconPickerProps extends ContentProps {
  /** The pill the picker opens under. */
  anchor: HTMLElement;
}

/**
 * The icon and color picker of a panel, closed until {@link openIconPicker} opens it.
 * Render it outside the pill strip, so its events never reach a pill.
 */
export const IconPicker = ({ panel, anchor }: IconPickerProps): ReactElement => (
  <Dialog.Frame id={frameID(panel)} anchor={anchor}>
    <Dialog.Dialog className={CSS.B("panel-icon-picker")} aria-label="Panel icon">
      <Errors.SuspenseBoundary>
        <Content panel={panel} />
      </Errors.SuspenseBoundary>
    </Dialog.Dialog>
  </Dialog.Frame>
);
