// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { view } from "@synnaxlabs/client";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { View as PView } from "@synnaxlabs/pluto";
import { color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Appearance } from "@/platform/appearance";
import { CSS } from "@/platform/css";
import { type View } from "@/platform/view/context";

const frameID = (key: view.Key): string => CSS.B(`view-icon-picker-${key}`);

/**
 * Opens the {@link IconPicker} of the view.
 *
 * @throws {Error} if no icon picker for the view is mounted.
 */
export const openIconPicker = (key: view.Key): void => Dialog.open(frameID(key));

interface ContentProps {
  /** The view whose icon and color the picker sets. */
  viewKey: view.Key;
}

const Content = ({ viewKey }: ContentProps): ReactElement | null => {
  const item = List.useItem<view.Key, View>(viewKey);
  const { update: setFields } = PView.useSetFields();
  if (item == null) return null;
  const handleIconChange = (next?: string) =>
    setFields({
      key: viewKey,
      actions: view.set({ path: ["icon"], value: next ?? null }),
    });
  const handleColorChange = (next?: color.Color) =>
    setFields({
      key: viewKey,
      actions: view.set({
        path: ["color"],
        value: next == null ? null : color.hex(next),
      }),
    });
  return (
    <Appearance.Picker
      icon={item.icon}
      color={item.color}
      fallback={Icon.View}
      onIconChange={handleIconChange}
      onColorChange={handleColorChange}
    />
  );
};

export interface IconPickerProps extends ContentProps {
  /** The tab the picker opens under. */
  anchor: HTMLElement;
}

/**
 * The icon and color picker of a view, closed until {@link openIconPicker} opens it.
 * Render it in the frame of the views, outside their tabs.
 */
export const IconPicker = ({ viewKey, anchor }: IconPickerProps): ReactElement => (
  <Dialog.Frame id={frameID(viewKey)} anchor={anchor}>
    <Appearance.Dialog aria-label="View icon">
      <Content viewKey={viewKey} />
    </Appearance.Dialog>
  </Dialog.Frame>
);
