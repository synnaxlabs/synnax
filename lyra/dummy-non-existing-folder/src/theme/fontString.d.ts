import { type text } from "@synnaxlabs/x";
import { type Component } from "../component";
import { type ThemeSpec } from "./theme";
interface FontStringOptions {
    level: text.Level | Component.Size;
    weight?: text.Weight;
    code?: boolean;
}
export declare const fontString: (theme: ThemeSpec, { level, weight, code }: FontStringOptions) => string;
export {};
//# sourceMappingURL=fontString.d.ts.map