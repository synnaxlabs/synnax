// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label } from "@synnaxlabs/client";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { Access } from "@/access";
import { Fields } from "@/label/Fields";
import { type formSchema, useForm } from "@/label/queries";

interface EditViewProps {
  itemKey: label.Key;
}

const EditView = ({ itemKey }: EditViewProps): ReactElement => {
  const { form } = useForm({ query: { key: itemKey } });
  return (
    <Form.Form<typeof formSchema> {...form}>
      <Text.Text level="small" color={9}>
        Changes every use of this label.
      </Text.Text>
      <Fields />
    </Form.Form>
  );
};

/**
 * The context menu of an option in a label select, for a user who can update the
 * label. Its Edit item opens the editor of the label in place of the options. Each edit
 * saves at once.
 */
export const OptionMenu = ({
  keys: [key],
}: Menu.ContextMenuMenuProps): ReactElement | null => {
  const { open } = Dialog.useViews();
  const canUpdate = Access.useUpdateGranted(label.resourceID(key));
  if (!canUpdate) return null;
  return (
    <Menu.Menu gap="small">
      <Menu.Item
        itemKey="edit"
        onClick={() => open("Edit label", <EditView itemKey={key} />)}
      >
        <Icon.Edit />
        Edit
      </Menu.Item>
    </Menu.Menu>
  );
};
