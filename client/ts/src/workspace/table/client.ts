import { ontology } from "@/ontology";
import { sendRequired, UnaryClient } from "@synnaxlabs/freighter";
import { toArray, UnknownRecord } from "@synnaxlabs/x";
import { unknownRecordZ } from "@synnaxlabs/x/record";
import { z } from "zod";

export const keyZ = z.string().uuid();
export type Key = z.infer<typeof keyZ>;
export type Params = Key | Key[];

export const tableZ = z.object({
  key: z.string(),
  name: z.string(),
  data: unknownRecordZ.or(z.string().transform((s) => JSON.parse(s) as UnknownRecord)),
});

export type Table = z.infer<typeof tableZ>;

export const ONTOLOGY_TYPE: ontology.ResourceType = "table";

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key });

const RETRIEVE_ENDPOINT = "/workspace/table/retrieve";
const CREATE_ENDPOINT = "/workspace/table/create";
const RENAME_ENDPOINT = "/workspace/table/rename";
const SET_DATA_ENDPOINT = "/workspace/table/set-data";
const DELETE_ENDPOINT = "/workspace/table/delete";

export const newTableZ = tableZ.partial({ key: true }).transform((p) => ({
  ...p,
  data: JSON.stringify(p.data),
}));

export type NewTable = z.input<typeof newTableZ>;

export const tableRemoteZ = z.object({
  key: z.string(),
  name: z.string(),
  data: z.string().transform((s) => JSON.parse(s) as UnknownRecord),
});

const retrieveReqZ = z.object({ keys: z.string().array() });
const createReqZ = z.object({ workspace: z.string(), tables: newTableZ.array() });
const renameReqZ = z.object({ key: z.string(), name: z.string() });
const setDataReqZ = z.object({ key: z.string(), data: z.string() });
const deleteReqZ = z.object({ keys: z.string().array() });

const retrieveResZ = z.object({ tables: tableRemoteZ.array() });
const createResZ = z.object({ tables: tableRemoteZ.array() });
const emptyResZ = z.object({});

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(workspace: string, table: NewTable): Promise<Table>;
  async create(workspace: string, tables: NewTable[]): Promise<Table[]>;
  async create(
    workspace: string,
    tables: NewTable | NewTable[],
  ): Promise<Table | Table[]> {
    const isMany = Array.isArray(tables);
    const normalized = toArray(tables);
    const res = await sendRequired(
      this.client,
      CREATE_ENDPOINT,
      { workspace, tables: normalized },
      createReqZ,
      createResZ,
    );
    return isMany ? res.tables : res.tables[0];
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

  async retrieve(key: Key): Promise<Table>;
  async retrieve(keys: Key[]): Promise<Table[]>;
  async retrieve(keys: Params): Promise<Table | Table[]> {
    const isMany = Array.isArray(keys);
    const normalized = toArray(keys);
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      { keys: normalized },
      retrieveReqZ,
      retrieveResZ,
    );
    return isMany ? res.tables : res.tables[0];
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
}
