// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label } from "@synnaxlabs/client";
import { Access, Dialog, Label } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Button } from "@/platform/button";
import { useEditModal } from "@/platform/label/useEditModal";

export interface SelectSingleProps extends Label.SelectSingleProps {}

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
  Access.useCreateGranted(label.TYPE_ONTOLOGY_ID) ? <CreateButton /> : undefined;

export const SelectSingle = (props: SelectSingleProps) => (
  <Label.SelectSingle {...props} footer={useCreateFooter()} />
);

export const SelectMultiple = (props: Label.SelectMultipleProps) => (
  <Label.SelectMultiple {...props} footer={useCreateFooter()} />
);
