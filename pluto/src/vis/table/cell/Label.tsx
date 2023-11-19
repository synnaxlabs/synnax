import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Text } from "@/text";
import { type Theming } from "@/theming";
import { type Spec, type CellProps, type FormProps } from "@/vis/table/cell/element";

import "@/vis/table/cell/Label.css";

export interface LabelProps extends Omit<Text.EditableProps, "onChange"> {}

const Cell = ({
  onChange,
  className,
  onSelect,
  ...props
}: CellProps<LabelProps>): ReactElement => {
  return (
    <td className={CSS(CSS.BE("table", "label"))} onDoubleClick={onSelect}>
      <Text.Editable
        {...props}
        onChange={(v) => {
          console.log(v);
          onChange({ ...props, value: v });
        }}
      />
    </td>
  );
};

const Form = ({ value, onChange }: FormProps<LabelProps>): ReactElement => {
  return <h1>Hello</h1>;
};

export const initialProps = (t: Theming.Theme): LabelProps => ({
  value: "Label",
  level: "p" as "h1",
});

export const LabelSpec: Spec<LabelProps> = {
  type: "label",
  title: "Label",
  initialProps,
  Cell,
  Form,
};
