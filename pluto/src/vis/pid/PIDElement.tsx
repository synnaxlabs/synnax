import { FC } from "react";

import { XY } from "@synnaxlabs/x";

import { InputControl } from "@/core";

export type PIDElementProps<P extends unknown = unknown> = P & {
  position: XY;
  selected: boolean;
  editable: boolean;
  onChange: (props: P) => void;
};

export interface PIDElementFormProps<P extends unknown = unknown>
  extends InputControl<P> {}

export interface PIDElementSpec<P extends unknown = unknown> {
  type: string;
  title: string;
  initialProps: P;
  Element: FC<PIDElementProps<P>>;
  Form: FC<PIDElementFormProps<P>>;
  Preview: FC<null>;
}
