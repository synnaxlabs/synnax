// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog, Label } from "@synnaxlabs/pluto";

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
    <Button.Create size="small" onClick={onClick}>
      New Label
    </Button.Create>
  );
};

export const SelectSingle = (props: SelectSingleProps) => (
  <Label.SelectSingle {...props} footer={<CreateButton />} />
);

export const SelectMultiple = (props: Label.SelectMultipleProps) => (
  <Label.SelectMultiple {...props} footer={<CreateButton />} />
);
