import "./Segmented.css";
import { type schematic } from "@synnaxlabs/client";
import { xy } from "@synnaxlabs/x";
import { type FC } from "react";
import { type Base } from "../base";
import { type Config } from "./config";
import { type Spec } from "../../spec";
export interface PathProps extends Omit<Base.BaseProps, "path" | "points"> {
    points: xy.XY[];
    crossings: xy.XY[];
}
export declare const createSpec: <V extends schematic.EdgeConfigType>(variant: V, name: string, path: FC<PathProps>) => Spec<V, Config<V>>;
//# sourceMappingURL=Segmented.d.ts.map