// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Dialog, Icon, Label } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { EmptyAction } from "@/components";
import { EDIT_LAYOUT } from "@/label/Edit";
import { Layout } from "@/layout";

export interface SelectSingleProps extends Label.SelectSingleProps {}

const useAdd = (): (() => void) => {
  const placeLayout = Layout.usePlacer();
  const { close } = Dialog.useContext();
  return () => {
    close();
    placeLayout(EDIT_LAYOUT);
  };
};

const SelectEmptyContent = (): ReactElement => {
  const add = useAdd();
  return (
    <EmptyAction message="No labels created." action="Create a label" onClick={add} />
  );
};

export const SelectSingle = (props: SelectSingleProps) => (
  <Label.SelectSingle
    emptyContent={<SelectEmptyContent />}
    {...props}
    actions={<AddButton />}
  />
);

const AddButton = () => {
  const onClick = useAdd();
  return (
    <Button.Button onClick={onClick} variant="outlined" contrast={3}>
      <Icon.Add />
    </Button.Button>
  );
};

export const SelectMultiple = (props: Label.SelectMultipleProps) => (
  <Label.SelectMultiple
    emptyContent={<SelectEmptyContent />}
    {...props}
    actions={<AddButton />}
  />
);
