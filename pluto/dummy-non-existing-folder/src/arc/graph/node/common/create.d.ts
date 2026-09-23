import { type FC } from "react";
import z from "zod/v4";
import { type Spec } from "../types/spec";
export interface OperatorParams<T extends string> {
    key: T;
    name: string;
    Symbol: FC;
}
export declare const createOperator: <T extends string>({ key, name, Symbol, }: OperatorParams<T>) => {
    configZ: z.ZodObject<{
        type: z.ZodLiteral<T>;
    }, z.core.$strip>;
    spec: Spec<T, {
        type: T;
    }>;
};
//# sourceMappingURL=create.d.ts.map