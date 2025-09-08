import { type record, type xy } from "@synnaxlabs/x";
import { type FC } from "react";

import { type Theming } from "@/theming";

export type PreviewProps<P extends object = record.Unknown> = P & {
  scale?: number;
};

export type SymbolProps<P extends object = record.Unknown> = P & {
  symbolKey?: string;
  position?: xy.XY;
  aetherKey?: string;
  selected?: boolean;
  draggable?: boolean;
  onChange?: (value: Partial<P>) => void;
  scale?: number;
};

export interface Spec<P extends object = record.Unknown> {
  key: string;
  name: string;
  Form: FC<{}>;
  Symbol: FC<SymbolProps<P>>;
  defaultProps: (t: Theming.Theme) => P;
  Preview: FC<PreviewProps<P>>;
  zIndex: number;
}
