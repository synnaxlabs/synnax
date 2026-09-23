import { type z } from "zod";
import { setpoint } from "./aether";
export interface UseProps extends Pick<z.input<typeof setpoint.stateZ>, "sink"> {
    aetherKey: string;
}
export interface UseReturn {
    set: (value: number) => void;
}
export declare const use: ({ aetherKey, sink }: UseProps) => UseReturn;
//# sourceMappingURL=use.d.ts.map