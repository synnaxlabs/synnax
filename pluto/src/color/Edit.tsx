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
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import type { ReactElement } from "react";

import { DarkInput, Editor, LightInput } from "@/color/Editor";
import { formPath } from "@/color/queries";

interface EditViewProps {
  itemKey: color.Key;
}

/** The named color editor of Settings, laid out for the inside of a picker. */
const EditView = ({ itemKey }: EditViewProps): ReactElement => (
  <Editor itemKey={itemKey}>
    <Text.Text level="small" color={9}>
      Changes every use of this color.
    </Text.Text>
    <Form.Field<string> path={formPath.name} label="Name" padHelpText={false}>
      {(p) => <Input.Text {...p} placeholder="Color name" />}
    </Form.Field>
    <Flex.Box x gap="large">
      <Input.Item label="Light" padHelpText={false}>
        <LightInput />
      </Input.Item>
      <Input.Item label="Dark" padHelpText={false}>
        <DarkInput />
      </Input.Item>
    </Flex.Box>
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
