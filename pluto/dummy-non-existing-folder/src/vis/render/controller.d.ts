import { type aether } from "../../aether/aether";
export interface Requestor {
    (reason: string): void;
}
export type RenderReason = "layout" | "data" | "tool";
export declare const control: (ctx: aether.Context, f: Requestor) => void;
export declare const useRequestor: (ctx: aether.Context) => Requestor;
export declare const useOptionalRequestor: (ctx: aether.Context) => Requestor | null;
export declare const request: (ctx: aether.Context, reason: RenderReason) => void;
//# sourceMappingURL=controller.d.ts.map