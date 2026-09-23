import { z } from "zod";
export type Primitive = string | number | boolean | null;
export type PrimitiveTypeName = "string" | "number" | "boolean" | "null";
export declare const primitiveZ: z.ZodUnion<readonly [z.ZodString, z.ZodNumber, z.ZodBoolean, z.ZodNull]>;
export declare const detectType: (value: Primitive) => PrimitiveTypeName;
export declare const ZERO_VALUES: Record<PrimitiveTypeName, Primitive>;
//# sourceMappingURL=primitive.d.ts.map