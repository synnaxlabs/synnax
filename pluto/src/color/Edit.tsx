// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Color } from "@synnaxlabs/lyra/color";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { Access } from "@/access";
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
      {(p) => <Input.Text {...p} placeholder="Color name" onlyChangeOnBlur />}
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

interface EditButtonProps {
  named: Color.Named;
}

const EditButton = ({ named }: EditButtonProps): ReactElement | null => {
  const { openView } = Color.usePicker();
  const canUpdate = Access.useUpdateGranted(color.resourceID(named.key));
  if (!canUpdate) return null;
  return (
    <Button.Button
      variant="text"
      size="tiny"
      square
      textColor={9}
      tooltip="Edit color"
      tooltipLocation="right"
      aria-label={`Edit ${named.name}`}
      onClick={() => openView("Edit color", <EditView itemKey={named.key} />)}
    >
      <Icon.Edit />
    </Button.Button>
  );
};

/**
 * Follows the name in a palette title of a color picker. It opens the named color
 * editor inside the picker, for a user who can update the color.
 */
export const EditAction = (): ReactElement | null => {
  const { shown } = Color.usePalette();
  return shown == null ? null : <EditButton named={shown} />;
};
