import { type z } from "zod";
import { toggle } from "./aether";
export interface UseProps extends Pick<z.input<typeof toggle.toggleStateZ>, "source" | "sink" | "stalenessTimeout"> {
    aetherKey: string;
}
export interface UseReturn extends Pick<z.infer<typeof toggle.toggleStateZ>, "enabled" | "stale"> {
    toggle: () => void;
}
export declare const use: ({ aetherKey, source, sink, stalenessTimeout, }: UseProps) => UseReturn;
//# sourceMappingURL=use.d.ts.map