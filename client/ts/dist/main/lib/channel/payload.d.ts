import { z } from 'zod';
import { DataType, Density, Rate } from '../telem';
export declare const channelPayloadSchema: z.ZodObject<{
    rate: z.ZodEffects<z.ZodNumber, Rate, number>;
    dataType: z.ZodEffects<z.ZodString, DataType, string>;
    key: z.ZodOptional<z.ZodDefault<z.ZodString>>;
    name: z.ZodOptional<z.ZodDefault<z.ZodString>>;
    nodeId: z.ZodOptional<z.ZodDefault<z.ZodNumber>>;
    density: z.ZodOptional<z.ZodEffects<z.ZodDefault<z.ZodNumber>, Density, number | undefined>>;
}, "strip", z.ZodTypeAny, {
    key?: string | undefined;
    density?: Density | undefined;
    name?: string | undefined;
    nodeId?: number | undefined;
    rate: Rate;
    dataType: DataType;
}, {
    key?: string | undefined;
    density?: number | undefined;
    name?: string | undefined;
    nodeId?: number | undefined;
    rate: number;
    dataType: string;
}>;
export declare type ChannelPayload = z.infer<typeof channelPayloadSchema>;
