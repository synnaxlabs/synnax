import { UnaryClient } from '@synnaxlabs/freighter';
import { z } from 'zod';
import Transport from '../transport';
import {
  OntologyID,
  OntologyResource,
  ontologyResourceSchema,
} from './payload';

const requestSchema = z.object({
  ids: z.string().array(),
  children: z.boolean().optional(),
  parents: z.boolean().optional(),
});

type Request = z.infer<typeof requestSchema>;

const responseSchema = z.object({
  resources: ontologyResourceSchema.array(),
});

export default class Retriever {
  private static ENDPOINT = '/ontology/retrieve';
  private client: UnaryClient;

  constructor(transport: Transport) {
    this.client = transport.getClient();
  }

  async execute(request: Request): Promise<OntologyResource[]> {
    const [res, err] = await this.client.send(
      Retriever.ENDPOINT,
      request,
      responseSchema
    );
    if (err) throw err;
    return res?.resources as OntologyResource[];
  }

  async retrieve(id: OntologyID): Promise<OntologyResource> {
    return (await this.execute({ ids: [id.toString()] }))[0];
  }

  async retrieveMany(...ids: OntologyID[]): Promise<OntologyResource[]> {
    return await this.execute({ ids: ids.map((id) => id.toString()) });
  }

  async retrieveChildren(...ids: OntologyID[]): Promise<OntologyResource[]> {
    return await this.execute({
      ids: ids.map((id) => id.toString()),
      children: true,
    });
  }

  async retrieveParents(...ids: OntologyID[]): Promise<OntologyResource[]> {
    return await this.execute({
      ids: ids.map((id) => id.toString()),
      parents: true,
    });
  }
}
