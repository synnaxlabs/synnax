import { type ReactElement } from "react";

import { Text } from "@/text";
import { type Theming } from "@/theming";
import {
  type Spec,
  type ElementProps,
  type FormProps,
} from "@/vis/table/element/element";

export interface LabelProps extends Omit<Text.EditableProps, "onChange"> {}

const Element = ({ onChange, ...props }: ElementProps<LabelProps>): ReactElement => {
  return (
    <Text.Editable {...props} onChange={(v) => onChange({ ...props, value: v })} />
  );
};

const Form = ({ value, onChange }: FormProps<LabelProps>): ReactElement => {
  return <></>;
};

export const initialProps = (t: Theming.Theme): LabelProps => ({
  value: "",
  level: "p" as "h1",
});

export const LabelSpec: Spec<LabelProps> = {
  type: "label",
  title: "Label",
  initialProps,
  Element,
  Form,
};
