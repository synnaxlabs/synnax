import { type FC } from "react";

import { type Diagram } from "@/vis/diagram";

export interface EdgeProps<Config extends object = object> extends Diagram.EdgeProps {
  onChange: (p: Partial<Config>) => void;
  config: Config;
}

export type Edge<Config extends object = object> = FC<EdgeProps<Config>>;

export interface Spec<Variant extends string = string, P extends object = object> {
  key: Variant;
  name: string;
  Form: FC;
  Edge: Edge<P>;
  defaultConfig: () => P;
}
