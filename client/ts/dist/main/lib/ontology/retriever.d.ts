import { z } from 'zod';
import Transport from '../transport';
import { OntologyID, OntologyResource } from './payload';
declare const requestSchema: z.ZodObject<{
    ids: z.ZodArray<z.ZodString, "many">;
    children: z.ZodOptional<z.ZodBoolean>;
    parents: z.ZodOptional<z.ZodBoolean>;
}, "strip", z.ZodTypeAny, {
    children?: boolean | undefined;
    parents?: boolean | undefined;
    ids: string[];
}, {
    children?: boolean | undefined;
    parents?: boolean | undefined;
    ids: string[];
}>;
declare type Request = z.infer<typeof requestSchema>;
export default class Retriever {
    private static ENDPOINT;
    private client;
    constructor(transport: Transport);
    execute(request: Request): Promise<OntologyResource[]>;
    retrieve(id: OntologyID): Promise<OntologyResource>;
    retrieveMany(...ids: OntologyID[]): Promise<OntologyResource[]>;
    retrieveChildren(...ids: OntologyID[]): Promise<OntologyResource[]>;
    retrieveParents(...ids: OntologyID[]): Promise<OntologyResource[]>;
}
export {};
