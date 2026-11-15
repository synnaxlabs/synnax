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
import { useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { Access, Label } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Button } from "@/platform/button";
import { useCreateModal } from "@/platform/label/useCreateModal";

interface CreateButtonProps {
  onCreate: (key: label.Key) => void;
}

const CreateButton = ({ onCreate }: CreateButtonProps): ReactElement => {
  const create = useCreateModal();
  const { close } = Dialog.useContext();
  const handleClick = (): void => {
    close();
    void create().then((key) => {
      if (key != null) onCreate(key);
    });
  };
  return (
    <Button.CreateListItem size="small" onClick={handleClick}>
      Create label
    </Button.CreateListItem>
  );
};

/** Picks labels. A user who can create labels can also create one, which it adds. */
export const SelectMultiple = ({
  value,
  onChange,
  ...rest
}: Label.SelectMultipleProps): ReactElement => {
  const canCreate = Access.useCreateGranted(label.TYPE_RESOURCE_ID);
  // The create dialog can stay open while the value changes.
  const valueRef = useSyncedRef(value);
  // The select moves its border onto a wrapper once it has a footer, so a user who
  // cannot create labels gets no footer rather than an empty one.
  const footer = canCreate ? (
    <CreateButton
      onCreate={(key) => onChange([...valueRef.current, key], { clicked: key })}
    />
  ) : undefined;
  return (
    <Label.SelectMultiple {...rest} value={value} onChange={onChange} footer={footer} />
  );
};
