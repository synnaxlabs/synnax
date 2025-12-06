// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { Access, Button, Dialog, Icon, Status } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { EmptyAction } from "@/components";
import { Layout } from "@/layout";
import { CREATE_LAYOUT } from "@/status/Create";

export const useCreate = (): (() => void) => {
  const placeLayout = Layout.usePlacer();
  const { close } = Dialog.useContext();
  return () => {
    close();
    placeLayout(CREATE_LAYOUT);
  };
};

export const SelectEmptyContent = (): ReactElement => {
  const add = useCreate();
  const canEdit = Access.useEditGranted(status.TYPE_ONTOLOGY_ID);
  return (
    <EmptyAction
      message="Non statuses created."
      action={canEdit ? "Create a Status" : undefined}
      onClick={canEdit ? add : undefined}
    />
  );
};

export const CreateButton = () => {
  const add = useCreate();
  const canEdit = Access.useEditGranted(status.TYPE_ONTOLOGY_ID);
  if (!canEdit) return null;
  return (
    <Button.Button onClick={add} title="Create a Status">
      <Icon.Add />
    </Button.Button>
  );
};

export interface SelectSingleProps extends Status.SelectProps {}

export const SelectSingle = (props: SelectSingleProps) => (
  <Status.Select
    emptyContent={<SelectEmptyContent />}
    actions={<CreateButton />}
    {...props}
  />
);
