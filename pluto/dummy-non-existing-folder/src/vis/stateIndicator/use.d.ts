import { type z } from "zod";
import { stateIndicator } from "./aether";
export interface UseProps extends Pick<z.input<typeof stateIndicator.stateZ>, "source" | "options" | "stalenessTimeout"> {
    aetherKey: string;
}
export interface UseReturn extends Pick<z.infer<typeof stateIndicator.stateZ>, "key" | "stale"> {
}
export declare const use: ({ aetherKey, source, options, stalenessTimeout, }: UseProps) => UseReturn;
//# sourceMappingURL=use.d.ts.map