import { type optional } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { Aether } from "../../aether";
import { line } from "./aether";
export interface LineProps extends optional.Optional<Omit<line.State, "key">, "strokeWidth">, Aether.ComponentProps {
}
export declare const Line: import("react").MemoExoticComponent<({ aetherKey, ...rest }: LineProps) => ReactElement | null>;
//# sourceMappingURL=Line.d.ts.map