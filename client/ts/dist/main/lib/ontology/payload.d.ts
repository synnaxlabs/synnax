import { z } from 'zod';
export declare enum OntologyResourceType {
    Builtin = "builtin",
    Cluster = "cluster",
    Channel = "channel",
    Node = "node"
}
export declare const ontologyIdSchema: z.ZodObject<{
    type: z.ZodNativeEnum<typeof OntologyResourceType>;
    key: z.ZodString;
}, "strip", z.ZodTypeAny, {
    type: OntologyResourceType;
    key: string;
}, {
    type: OntologyResourceType;
    key: string;
}>;
export declare class OntologyID {
    type: OntologyResourceType;
    key: string;
    constructor(type: OntologyResourceType, key: string);
    toString(): string;
    static parseString(str: string): OntologyID;
}
export declare const OntologyRoot: OntologyID;
export declare const ontologySchemaFieldSchema: z.ZodObject<{
    type: z.ZodNumber;
}, "strip", z.ZodTypeAny, {
    type: number;
}, {
    type: number;
}>;
export declare type OntologySchemaField = z.infer<typeof ontologySchemaFieldSchema>;
export declare const ontologySchemaSchema: z.ZodObject<{
    type: z.ZodNativeEnum<typeof OntologyResourceType>;
    fields: z.ZodRecord<z.ZodString, z.ZodObject<{
        type: z.ZodNumber;
    }, "strip", z.ZodTypeAny, {
        type: number;
    }, {
        type: number;
    }>>;
}, "strip", z.ZodTypeAny, {
    type: OntologyResourceType;
    fields: Record<string, {
        type: number;
    }>;
}, {
    type: OntologyResourceType;
    fields: Record<string, {
        type: number;
    }>;
}>;
export declare type OntologySchema = z.infer<typeof ontologySchemaSchema>;
export declare const ontologyResourceSchema: z.ZodObject<{
    id: z.ZodEffects<z.ZodObject<{
        type: z.ZodNativeEnum<typeof OntologyResourceType>;
        key: z.ZodString;
    }, "strip", z.ZodTypeAny, {
        type: OntologyResourceType;
        key: string;
    }, {
        type: OntologyResourceType;
        key: string;
    }>, OntologyID, {
        type: OntologyResourceType;
        key: string;
    }>;
    entity: z.ZodObject<{
        schema: z.ZodObject<{
            type: z.ZodNativeEnum<typeof OntologyResourceType>;
            fields: z.ZodRecord<z.ZodString, z.ZodObject<{
                type: z.ZodNumber;
            }, "strip", z.ZodTypeAny, {
                type: number;
            }, {
                type: number;
            }>>;
        }, "strip", z.ZodTypeAny, {
            type: OntologyResourceType;
            fields: Record<string, {
                type: number;
            }>;
        }, {
            type: OntologyResourceType;
            fields: Record<string, {
                type: number;
            }>;
        }>;
        name: z.ZodString;
        data: z.ZodRecord<z.ZodString, z.ZodUnknown>;
    }, "strip", z.ZodTypeAny, {
        data: Record<string, unknown>;
        schema: {
            type: OntologyResourceType;
            fields: Record<string, {
                type: number;
            }>;
        };
        name: string;
    }, {
        data: Record<string, unknown>;
        schema: {
            type: OntologyResourceType;
            fields: Record<string, {
                type: number;
            }>;
        };
        name: string;
    }>;
}, "strip", z.ZodTypeAny, {
    id: OntologyID;
    entity: {
        data: Record<string, unknown>;
        schema: {
            type: OntologyResourceType;
            fields: Record<string, {
                type: number;
            }>;
        };
        name: string;
    };
}, {
    id: {
        type: OntologyResourceType;
        key: string;
    };
    entity: {
        data: Record<string, unknown>;
        schema: {
            type: OntologyResourceType;
            fields: Record<string, {
                type: number;
            }>;
        };
        name: string;
    };
}>;
export declare type OntologyResource = z.infer<typeof ontologyResourceSchema>;
