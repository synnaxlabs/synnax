import { type ReactElement } from "react";
import { type Aether } from "../aether";
import { Line as Base } from "../vis/line";
export interface LineProps extends Base.LineProps, Aether.ComponentProps {
    legendGroup: string;
}
export declare const Line: ({ aetherKey, color, label, legendGroup, visible, ...rest }: LineProps) => ReactElement;
//# sourceMappingURL=Line.d.ts.map