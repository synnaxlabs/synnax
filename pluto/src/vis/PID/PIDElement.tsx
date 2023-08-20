import { FC } from "react";

import { XY } from "@synnaxlabs/x";

import { InputControl } from "@/core";

export type StatefulPIDElementProps<P extends object = {}> = P & {
  position: XY;
  selected: boolean;
  editable: boolean;
  onChange: (props: P) => void;
};

export interface PIDElementFormProps<P extends object = {}> extends InputControl<P> {}

export interface PIDElementSpec<P extends object = {}> {
  type: string;
  title: string;
  initialProps: P;
  Element: FC<StatefulPIDElementProps<P>>;
  Form: FC<PIDElementFormProps<P>>;
  Preview: FC<{}>;
}
