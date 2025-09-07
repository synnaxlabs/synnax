import { type record, type xy } from "@synnaxlabs/x";
import { type FC } from "react";

import { type Theming } from "@/theming";

export type SymbolProps<P extends object = record.Unknown> = P & {
  symbolKey: string;
  position: xy.XY;
  aetherKey: string;
  selected: boolean;
  draggable: boolean;
  onChange: (value: Partial<P>) => void;
};

export interface Spec<P extends object> {
  key: string;
  name: string;
  Form: FC<{}>;
  Symbol: FC<SymbolProps<P>>;
  defaultProps: (t: Theming.Theme) => P;
  Preview: FC<SymbolProps<P>>;
  zIndex: number;
}
