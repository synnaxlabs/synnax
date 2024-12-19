import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { toArray, type UnknownRecord } from "@synnaxlabs/x";
import { unknownRecordZ } from "@synnaxlabs/x/record";
import { type AsyncTermSearcher } from "@synnaxlabs/x/search";
import { z } from "zod";

import { ontology } from "@/ontology";
import { nullableArrayZ } from "@/util/zod";
import { linePlot } from "@/workspace/lineplot";
import { log } from "@/workspace/log";
import { schematic } from "@/workspace/schematic";
import { table } from "@/workspace/table";

export const keyZ = z.string().uuid();
export type Key = z.infer<typeof keyZ>;
export type Params = Key | Key[];

// --- VERY IMPORTANT ---
// Synnax's encoders (in the binary package inside x) automatically convert the case
// of keys in objects to snake_case and back to camelCase when encoding and decoding
// respectively. This is done to ensure that the keys are consistent across all
// languages and platforms. Sometimes workspaces have keys that are uuids, which have
// dashes, and those get messed up. So we just use regular JSON for workspaces.
const parse = (s: string): UnknownRecord => JSON.parse(s) as UnknownRecord;

export const workspaceZ = z.object({
  key: z.string(),
  name: z.string(),
  layout: unknownRecordZ.or(z.string().transform((s) => parse(s) as UnknownRecord)),
});

export type Workspace = z.infer<typeof workspaceZ>;

export const ONTOLOGY_TYPE: ontology.ResourceType = "workspace";

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key });

const RETRIEVE_ENDPOINT = "/workspace/retrieve";
const CREATE_ENDPOINT = "/workspace/create";
const RENAME_ENDPOINT = "/workspace/rename";
const SET_LAYOUT_ENDPOINT = "/workspace/set-layout";
const DELETE_ENDPOINT = "/workspace/delete";

export const newWorkspaceZ = workspaceZ.partial({ key: true }).transform((p) => ({
  ...p,
  layout: JSON.stringify(p.layout),
}));

export const workspaceRemoteZ = workspaceZ.omit({ layout: true }).extend({
  layout: z.string().transform((s) => parse(s) as UnknownRecord),
});

export type NewWorkspace = z.input<typeof newWorkspaceZ>;

const retrieveReqZ = z.object({
  keys: z.string().array().optional(),
  search: z.string().optional(),
  author: z.string().uuid().optional(),
  offset: z.number().optional(),
  limit: z.number().optional(),
});
const createReqZ = z.object({ workspaces: newWorkspaceZ.array() });
const renameReqZ = z.object({ key: z.string(), name: z.string() });
const setLayoutReqZ = z.object({ key: z.string(), layout: z.string() });
const deleteReqZ = z.object({ keys: z.string().array() });

const retrieveResZ = z.object({ workspaces: nullableArrayZ(workspaceZ) });
const createResZ = z.object({ workspaces: workspaceRemoteZ.array() });
const emptyResZ = z.object({});

export class Client implements AsyncTermSearcher<string, Key, Workspace> {
  readonly type = "workspace";
  readonly schematic: schematic.Client;
  readonly linePlot: linePlot.Client;
  readonly log: log.Client;
  readonly table: table.Client;
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
    this.schematic = new schematic.Client(client);
    this.linePlot = new linePlot.Client(client);
    this.log = new log.Client(client);
    this.table = new table.Client(client);
  }

  async create(workspace: NewWorkspace): Promise<Workspace>;
  async create(workspaces: NewWorkspace[]): Promise<Workspace[]>;
  async create(
    workspaces: NewWorkspace | NewWorkspace[],
  ): Promise<Workspace | Workspace[]> {
    const isMany = Array.isArray(workspaces);
    const normalized = toArray(workspaces);
    const res = await sendRequired(
      this.client,
      CREATE_ENDPOINT,
      { workspaces: normalized },
      createReqZ,
      createResZ,
    );
    return isMany ? res.workspaces : res.workspaces[0];
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

  async setLayout(key: Key, layout: UnknownRecord): Promise<void> {
    await sendRequired(
      this.client,
      SET_LAYOUT_ENDPOINT,
      { key, layout: JSON.stringify(layout) },
      setLayoutReqZ,
      emptyResZ,
    );
  }

  async retrieve(key: Key): Promise<Workspace>;
  async retrieve(keys: Key[]): Promise<Workspace[]>;
  async retrieve(keys: Params): Promise<Workspace | Workspace[]> {
    const isMany = Array.isArray(keys);
    const normalized = toArray(keys);
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      { keys: normalized },
      retrieveReqZ,
      retrieveResZ,
    );
    return isMany ? res.workspaces : res.workspaces[0];
  }

  async retrieveByAuthor(author: string): Promise<Workspace[]> {
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      { author },
      retrieveReqZ,
      retrieveResZ,
    );
    return res.workspaces;
  }

  async search(term: string): Promise<Workspace[]> {
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      { search: term },
      retrieveReqZ,
      retrieveResZ,
    );
    return res.workspaces;
  }

  async page(offset: number, limit: number): Promise<Workspace[]> {
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      { offset, limit },
      retrieveReqZ,
      retrieveResZ,
    );
    return res.workspaces;
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
