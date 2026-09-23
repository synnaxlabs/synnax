import { type z } from "zod";
import { stringValue } from "./aether";
export interface UseProps extends Pick<z.input<typeof stringValue.stateZ>, "telem" | "stalenessTimeout"> {
    aetherKey: string;
}
export interface UseReturn extends Pick<z.infer<typeof stringValue.stateZ>, "value" | "stale"> {
}
export declare const use: ({ aetherKey, telem, stalenessTimeout }: UseProps) => UseReturn;
//# sourceMappingURL=use.d.ts.map