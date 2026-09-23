import { input } from "./aether";
export interface UseProps extends Pick<input.State, "sink"> {
    aetherKey: string;
}
export interface UseReturn {
    set: (value: string) => void;
}
export declare const use: ({ aetherKey, sink }: UseProps) => UseReturn;
//# sourceMappingURL=use.d.ts.map