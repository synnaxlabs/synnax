import { type ReactElement } from "react";

import { Remote } from "@/telem/remote";
import { type Theming } from "@/theming";
import { Table } from "@/vis/table";
import {
  type Spec,
  type FormProps,
  type ElementProps,
} from "@/vis/table/element/element";

export interface ValueProps extends Omit<Table.StringTDProps, "telem"> {
  telem: Remote.NumericSourceProps;
}

const Element = ({
  onChange,
  telem: propsTelem,
  ...props
}: ElementProps<ValueProps>): ReactElement => {
  const telem = Remote.useNumericStringSource(propsTelem);
  return <Table.StringTD {...props} telem={telem} />;
};

const Form = ({ value, onChange }: FormProps<ValueProps>): ReactElement => {
  return (
    <Remote.NumericStringSourceForm
      value={value.telem}
      onChange={(p) => onChange({ ...value, telem: p })}
    />
  );
};

export const initialProps = (t: Theming.Theme): ValueProps => ({
  color: t.colors.gray.l8,
  telem: {
    channel: 0,
    units: "",
    precision: 2,
  },
});

export const ValueSpec: Spec<ValueProps> = {
  type: "value",
  title: "Value",
  initialProps,
  Element,
  Form,
};
