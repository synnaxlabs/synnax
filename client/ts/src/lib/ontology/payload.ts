import { z } from 'zod';

export enum OntologyResourceType {
  Builtin = 'builtin',
  Cluster = 'cluster',
  Channel = 'channel',
  Node = 'node',
}

export const ontologyIdSchema = z.object({
  type: z.nativeEnum(OntologyResourceType),
  key: z.string(),
});

export class OntologyID {
  type: OntologyResourceType;
  key: string;

  constructor(type: OntologyResourceType, key: string) {
    this.type = type;
    this.key = key;
  }

  toString(): string {
    return `${this.type}:${this.key}`;
  }

  static parseString(str: string): OntologyID {
    const [type, key] = str.split(':');
    return new OntologyID(type as OntologyResourceType, key);
  }
}

export const OntologyRoot = new OntologyID(
  OntologyResourceType.Builtin,
  'root'
);

export const ontologySchemaFieldSchema = z.object({
  type: z.number(),
});

export type OntologySchemaField = z.infer<typeof ontologySchemaFieldSchema>;

export const ontologySchemaSchema = z.object({
  type: z.nativeEnum(OntologyResourceType),
  fields: z.record(ontologySchemaFieldSchema),
});

export type OntologySchema = z.infer<typeof ontologySchemaSchema>;

export const ontologyResourceSchema = z.object({
  id: ontologyIdSchema.transform((id) => new OntologyID(id.type, id.key)),
  entity: z.object({
    schema: ontologySchemaSchema,
    name: z.string(),
    data: z.record(z.unknown()),
  }),
});

export type OntologyResource = z.infer<typeof ontologyResourceSchema>;
