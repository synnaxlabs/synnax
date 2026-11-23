// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { color } from "@synnaxlabs/client";
import { Color } from "@synnaxlabs/lyra/color";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import type { ReactElement } from "react";

import { Editor } from "@/color/Editor";
import { Fields } from "@/color/View";

interface EditViewProps {
  itemKey: color.Key;
}

/** Edits a named color from a view of the picker. Each edit saves at once. */
const EditView = ({ itemKey }: EditViewProps): ReactElement => (
  <Editor itemKey={itemKey}>
    <Fields note="Changes every use of this color." />
  </Editor>
);

/**
 * The context menu of a swatch in a color picker, for a user who can update its color.
 * Its Edit item opens the named color editor inside the picker.
 */
export const SwatchMenu = (): ReactElement | null => {
  const { swatch } = Color.usePalette();
  const { open } = Dialog.useViews();
  if (swatch == null) return null;
  return (
    <Menu.Menu gap="small">
      <Menu.Item
        itemKey="edit"
        onClick={() => open("Edit color", <EditView itemKey={swatch.key} />)}
      >
        <Icon.Edit />
        Edit
      </Menu.Item>
    </Menu.Menu>
  );
};
