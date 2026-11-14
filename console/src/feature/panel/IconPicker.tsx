// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Errors, Panel } from "@synnaxlabs/pluto";
import { type color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Appearance } from "@/platform/appearance";
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

const Content = ({ panel: key }: ContentProps): ReactElement => {
  Panel.useEnsure({ key });
  const icon = Panel.useIcon({ key });
  const color = Panel.useColor({ key });
  const { dispatch } = Panel.useDispatch();
  const handleColorChange = (next?: color.Color) =>
    dispatch({ key, actions: panel.setColor({ color: next }) });
  const handleIconChange = (next?: string) =>
    dispatch({ key, actions: panel.setIcon({ icon: next }) });
  return (
    <Appearance.Picker
      icon={icon}
      color={color}
      fallback={Icon.Panel}
      onIconChange={handleIconChange}
      onColorChange={handleColorChange}
    />
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
    <Appearance.Dialog aria-label="Panel icon">
      <Errors.SuspenseBoundary>
        <Content panel={panel} />
      </Errors.SuspenseBoundary>
    </Appearance.Dialog>
  </Dialog.Frame>
);
