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
import { Access, Label } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { Button } from "@/platform/button";
import { useEditModal } from "@/platform/label/useEditModal";

const useAdd = (): (() => void) => {
  const openEdit = useEditModal();
  const { close } = Dialog.useContext();
  return () => {
    close();
    openEdit();
  };
};

const CreateButton = () => {
  const onClick = useAdd();
  return (
    <Button.CreateListItem size="small" onClick={onClick}>
      New label
    </Button.CreateListItem>
  );
};

// The select moves its border onto a wrapper the moment it is handed a footer, so an
// ungranted subject must leave the prop undefined rather than render nothing inside it.
const useCreateFooter = (): ReactElement | undefined =>
  Access.useCreateGranted(label.TYPE_RESOURCE_ID) ? <CreateButton /> : undefined;

/**
 * Selects labels. Until the user first opens it, it shows only the labels of the value:
 * the list of all labels and its dialog mount on the first open, since a page can hold
 * one of these per item.
 */
export const SelectMultiple = (props: Label.SelectMultipleProps): ReactElement => {
  const [opened, setOpened] = useState(false);
  // Resolved before the first open, so the list does not mount again when it resolves.
  const footer = useCreateFooter();
  if (opened || props.visible === true)
    return <Label.SelectMultiple initialVisible={opened} {...props} footer={footer} />;
  return <Label.SelectMultipleTrigger {...props} onOpen={() => setOpened(true)} />;
};
