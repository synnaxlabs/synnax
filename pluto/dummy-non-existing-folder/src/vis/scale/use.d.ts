import { type z } from "zod";
import { scale } from "./aether";
export declare const gutter: typeof scale.gutter;
export interface UseProps extends z.input<typeof scale.Scale.z> {
    aetherKey: string;
}
export declare const use: ({ aetherKey, ...state }: UseProps) => void;
//# sourceMappingURL=use.d.ts.map