// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/label/Edit.css";

import { label } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { Access } from "@/access";
import { CSS } from "@/css";
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
      <Fields onlyChangeOnBlur />
    </Form.Form>
  );
};

interface EditButtonProps {
  itemKey: label.Key;
  name: string;
}

/**
 * Opens the editor of a label in place of the options around it, for a user who can
 * update the label. Each edit saves at once.
 */
export const EditButton = ({ itemKey, name }: EditButtonProps): ReactElement | null => {
  const { open } = Dialog.useViews();
  const canUpdate = Access.useUpdateGranted(label.resourceID(itemKey));
  if (!canUpdate) return null;
  return (
    <Button.Button
      variant="text"
      size="tiny"
      square
      textColor={9}
      tooltip="Edit label"
      tooltipLocation="right"
      aria-label={`Edit ${name}`}
      className={CSS.BE("label", "edit")}
      onClick={() => open("Edit label", <EditView itemKey={itemKey} />)}
    >
      <Icon.Edit />
    </Button.Button>
  );
};
