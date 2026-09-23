import { type FC } from "react";
/**
 * Memoizes a generic component, keeping its type parameters. React's own `memo`
 * resolves them against their constraints, so its result is callable at that one
 * instantiation only.
 *
 * @example export const Frame = Component.memo(BaseFrame);
 */
export declare const memo: <T extends FC<any>>(component: T) => T;
//# sourceMappingURL=memo.d.ts.map