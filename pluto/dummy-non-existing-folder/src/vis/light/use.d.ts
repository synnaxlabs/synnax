import { type z } from "zod";
import { light } from "./aether";
export interface UseProps extends Pick<z.input<typeof light.stateZ>, "source" | "stalenessTimeout"> {
    aetherKey: string;
}
export interface UseReturn extends Pick<z.infer<typeof light.stateZ>, "enabled" | "stale"> {
}
export declare const use: ({ aetherKey, source, stalenessTimeout }: UseProps) => UseReturn;
//# sourceMappingURL=use.d.ts.map