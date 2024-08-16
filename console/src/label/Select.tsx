import { Align, Button, Label, Status, Text } from "@synnaxlabs/pluto";
import { ReactElement } from "react";

import { createEditLayout } from "@/label/Edit";
import { Layout } from "@/layout";

export interface SelectSingleProps extends Label.SelectSingleProps {}

const SelectEmptyContent = (): ReactElement => {
  const placeLayout = Layout.usePlacer();
  return (
    <Align.Center style={{ height: 150 }} direction="x">
      <Status.Text variant="disabled" hideIcon>
        No Labels:
      </Status.Text>
      <Text.Link
        level="p"
        onClick={() => {
          placeLayout(createEditLayout());
        }}
      >
        Add a Label
      </Text.Link>
    </Align.Center>
  );
};

export const SelectSingle = (props: SelectSingleProps) => (
  <Label.SelectSingle emptyContent={<SelectEmptyContent />} {...props} />
);

export const SelectMultiple = (props: Label.SelectMultipleProps) => (
  <Label.SelectMultiple emptyContent={<SelectEmptyContent />} {...props} />
);
