import { z } from 'zod';
export declare const UserPayloadSchema: z.ZodObject<{
    key: z.ZodString;
    username: z.ZodString;
}, "strip", z.ZodTypeAny, {
    key: string;
    username: string;
}, {
    key: string;
    username: string;
}>;
export declare type UserPayload = z.infer<typeof UserPayloadSchema>;
