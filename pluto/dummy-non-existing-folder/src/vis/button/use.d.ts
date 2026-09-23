import { type z } from "zod";
import { button } from "./aether";
export type Mode = button.Mode;
export declare const MODES: readonly ["fire", "momentary", "pulse"];
export interface UseProps extends z.input<typeof button.buttonStateZ> {
    aetherKey: string;
}
export interface UseReturn {
    onClick: () => void;
    onMouseDown: () => void;
    onMouseUp: () => void;
}
export declare const use: ({ aetherKey, sink, mode }: UseProps) => UseReturn;
//# sourceMappingURL=use.d.ts.map