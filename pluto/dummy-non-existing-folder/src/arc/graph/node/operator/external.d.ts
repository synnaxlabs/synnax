import z from "zod/v4";
export declare const REGISTRY: {
    add: import("../external").Spec<"add", {
        type: "add";
    }>;
    subtract: import("../external").Spec<"subtract", {
        type: "subtract";
    }>;
    multiply: import("../external").Spec<"multiply", {
        type: "multiply";
    }>;
    divide: import("../external").Spec<"divide", {
        type: "divide";
    }>;
    gt: import("../external").Spec<"gt", {
        type: "gt";
    }>;
    lt: import("../external").Spec<"lt", {
        type: "lt";
    }>;
    eq: import("../external").Spec<"eq", {
        type: "eq";
    }>;
    ne: import("../external").Spec<"ne", {
        type: "ne";
    }>;
    ge: import("../external").Spec<"ge", {
        type: "ge";
    }>;
    le: import("../external").Spec<"le", {
        type: "le";
    }>;
    and: import("../external").Spec<"and", {
        type: "and";
    }>;
    or: import("../external").Spec<"or", {
        type: "or";
    }>;
    not: import("../external").Spec<"not", {
        type: "not";
    }>;
};
export declare const configZ: z.ZodDiscriminatedUnion<[z.ZodObject<{
    type: z.ZodLiteral<"add">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"subtract">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"multiply">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"divide">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"gt">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"lt">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"eq">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"ne">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"ge">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"le">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"and">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"or">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"not">;
}, z.core.$strip>], "type">;
//# sourceMappingURL=external.d.ts.map