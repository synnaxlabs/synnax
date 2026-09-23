import { box } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement } from "react";
import { Aether } from "../../aether";
export interface UseProps {
    aetherKey?: string;
    enabled?: boolean;
}
export interface UseReturn {
    erase: (region: box.Box) => void;
}
export declare const use: ({ aetherKey, enabled }: UseProps) => UseReturn;
export interface EraserProps extends PropsWithChildren, Aether.ComponentProps {
}
export declare const Eraser: ({ aetherKey, children }: EraserProps) => ReactElement;
//# sourceMappingURL=use.d.ts.map