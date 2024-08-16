import { Icon } from "@synnaxlabs/media";
import { Align, Button, Dialog, Label, Status, Text } from "@synnaxlabs/pluto";
import { ReactElement } from "react";

import { createEditLayout } from "@/label/Edit";
import { Layout } from "@/layout";

export interface SelectSingleProps extends Label.SelectSingleProps {}

const useAdd = (): (() => void) => {
  const placeLayout = Layout.usePlacer();
  const { close } = Dialog.useContext();
  return () => {
    close();
    placeLayout(createEditLayout());
  };
};

const SelectEmptyContent = (): ReactElement => {
  const add = useAdd();
  return (
    <Align.Center style={{ height: 150 }} direction="y">
      <Status.Text variant="disabled" hideIcon>
        No labels created.
      </Status.Text>
      <Text.Link level="p" onClick={add}>
        Create a Label
      </Text.Link>
    </Align.Center>
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
    <Button.Icon onClick={onClick}>
      <Icon.Add />
    </Button.Icon>
  );
};

export const SelectMultiple = (props: Label.SelectMultipleProps) => (
  <Label.SelectMultiple
    emptyContent={<SelectEmptyContent />}
    {...props}
    actions={<AddButton />}
  />
);
