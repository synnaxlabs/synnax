import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { toArray, type UnknownRecord } from "@synnaxlabs/x";
import { unknownRecordZ } from "@synnaxlabs/x/record";
import { z } from "zod";

import { ontology } from "@/ontology";
import { nullableArrayZ } from "@/util/zod";

export const keyZ = z.string().uuid();
export type Key = z.infer<typeof keyZ>;
export type Params = Key | Key[];

export const schematicZ = z.object({
  key: z.string(),
  name: z.string(),
  data: unknownRecordZ.or(z.string().transform((s) => JSON.parse(s) as UnknownRecord)),
  snapshot: z.boolean(),
});

export const schematicRemoteZ = z.object({
  key: z.string(),
  name: z.string(),
  snapshot: z.boolean(),
  data: z.string().transform((s) => JSON.parse(s) as UnknownRecord),
});

export type Schematic = z.infer<typeof schematicZ>;

export const ONTOLOGY_TYPE: ontology.ResourceType = "schematic";

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key });

const RETRIEVE_ENDPOINT = "/workspace/schematic/retrieve";
const CREATE_ENDPOINT = "/workspace/schematic/create";
const RENAME_ENDPOINT = "/workspace/schematic/rename";
const SET_DATA_ENDPOINT = "/workspace/schematic/set-data";
const DELETE_ENDPOINT = "/workspace/schematic/delete";
const COPY_ENDPOINT = "/workspace/schematic/copy";

export const newSchematicZ = schematicZ
  .partial({ key: true, snapshot: true })
  .transform((p) => ({
    ...p,
    data: JSON.stringify(p.data),
  }));

export type NewSchematic = z.input<typeof newSchematicZ>;

const retrieveReqZ = z.object({ keys: z.string().array() });
const createReqZ = z.object({
  workspace: z.string(),
  schematics: newSchematicZ.array(),
});
const renameReqZ = z.object({ key: z.string(), name: z.string() });
const setDataReqZ = z.object({ key: z.string(), data: z.string() });
const deleteReqZ = z.object({ keys: z.string().array() });
const copyReqZ = z.object({ key: z.string(), name: z.string(), snapshot: z.boolean() });

const retrieveResZ = z.object({ schematics: nullableArrayZ(schematicRemoteZ) });
const createResZ = z.object({ schematics: schematicRemoteZ.array() });
const copyResZ = z.object({ schematic: schematicZ });
const emptyResZ = z.object({});

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(workspace: string, schematic: NewSchematic): Promise<Schematic>;
  async create(workspace: string, schematics: NewSchematic[]): Promise<Schematic[]>;
  async create(
    workspace: string,
    schematics: NewSchematic | NewSchematic[],
  ): Promise<Schematic | Schematic[]> {
    const isMany = Array.isArray(schematics);
    const normalized = toArray(schematics);
    const res = await sendRequired(
      this.client,
      CREATE_ENDPOINT,
      { workspace, schematics: normalized },
      createReqZ,
      createResZ,
    );
    return isMany ? res.schematics : res.schematics[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    await sendRequired(
      this.client,
      RENAME_ENDPOINT,
      { key, name },
      renameReqZ,
      emptyResZ,
    );
  }

  async setData(key: Key, data: UnknownRecord): Promise<void> {
    await sendRequired(
      this.client,
      SET_DATA_ENDPOINT,
      { key, data: JSON.stringify(data) },
      setDataReqZ,
      emptyResZ,
    );
  }

  async retrieve(key: Key): Promise<Schematic>;
  async retrieve(keys: Key[]): Promise<Schematic[]>;
  async retrieve(keys: Params): Promise<Schematic | Schematic[]> {
    const isMany = Array.isArray(keys);
    const normalized = toArray(keys);
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      { keys: normalized },
      retrieveReqZ,
      retrieveResZ,
    );
    return isMany ? res.schematics : res.schematics[0];
  }

  async delete(key: Key): Promise<void>;
  async delete(keys: Key[]): Promise<void>;
  async delete(keys: Params): Promise<void> {
    const normalized = toArray(keys);
    await sendRequired(
      this.client,
      DELETE_ENDPOINT,
      { keys: normalized },
      deleteReqZ,
      emptyResZ,
    );
  }

  async copy(key: Key, name: string, snapshot: boolean): Promise<Schematic> {
    const res = await sendRequired(
      this.client,
      COPY_ENDPOINT,
      { key, name, snapshot },
      copyReqZ,
      copyResZ,
    );
    return res.schematic;
  }
}
