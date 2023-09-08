// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Handle, Position } from "reactflow";

import { Button as CoreButton } from "@/button";
import { Bool, Control } from "@/index";
import { Input } from "@/input";
import { Button } from "@/vis/button";
import { type Props, type FormProps, type Spec } from "@/vis/pid/element/element";

const { Left, Right, Top, Bottom } = Position;

interface ElementProps extends Omit<Button.ButtonProps, "children" | "sink"> {
  label: string;
  sink: Control.NumericSinkProps;
}

const Element = ({ label, sink, ...props }: Props<ElementProps>): ReactElement => {
  const sinkN = Control.useNumeric(sink);
  const sinkB = Bool.useNumericConverterSink({
    wrap: sinkN,
    truthy: 1,
    falsy: 0,
  });

  return (
    <>
      <Handle position={Left} id="a" type="source" />
      <Handle position={Right} id="b" type="source" />
      <Handle position={Top} id="c" type="source" />
      <Handle position={Bottom} id="d" type="source" />
      <Button.Button sink={sinkB} {...props}>
        {label}
      </Button.Button>
    </>
  );
};

const Form = ({ onChange, value }: FormProps<ElementProps>): ReactElement => {
  const handleLabelChange = (label: string): void => onChange({ ...value, label });

  return (
    <>
      <Input.Item<string>
        label="Label"
        onChange={handleLabelChange}
        value={value.label}
      />
      <Control.NumericSinkForm
        label="Trigger"
        onChange={(sink) => onChange({ ...value, sink })}
        value={value.sink}
      />
    </>
  );
};

const Preview = (): ReactElement => (
  <CoreButton.Button variant="filled">Button</CoreButton.Button>
);

const initialProps = (): ElementProps => ({
  label: "Button",
  sink: { channel: 0 },
});

export const ButtonSpec: Spec<ElementProps> = {
  type: "button",
  title: "Button",
  Preview,
  initialProps,
  Element,
  Form,
};
