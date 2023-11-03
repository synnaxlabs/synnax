import { type FC } from "react";

import { type UnknownRecord } from "@synnaxlabs/x";

import { type Input } from "@/input";
import { type Theming } from "@synnaxlabs/pluto/theming";

export type ElementProps<P extends object = UnknownRecord> = P & {
  onChange: (props: P) => void;
};

export interface FormProps<P extends object = UnknownRecord> extends Input.Control<P> {}

export interface Spec<P extends object = UnknownRecord> {
  type: string;
  title: string;
  initialProps: (theme: Theming.Theme) => P;
  Element: FC<ElementProps<P>>;
  Form: FC<FormProps<P>>;
}
