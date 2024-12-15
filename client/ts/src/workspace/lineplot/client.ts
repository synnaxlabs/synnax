import { ontology } from "@/ontology";
import { sendRequired, UnaryClient } from "@synnaxlabs/freighter";
import { breaker, TimeSpan, toArray, UnknownRecord } from "@synnaxlabs/x";
import { unknownRecordZ } from "@synnaxlabs/x/record";
import { z } from "zod";

export const keyZ = z.string().uuid();
export type Key = z.infer<typeof keyZ>;
export type Params = Key | Key[];

export const linePlotZ = z.object({
  key: z.string(),
  name: z.string(),
  data: unknownRecordZ.or(z.string().transform((s) => JSON.parse(s) as UnknownRecord)),
});

export type LinePlot = z.infer<typeof linePlotZ>;

export const ONTOLOGY_TYPE: ontology.ResourceType = "lineplot";

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key });

const RETRIEVE_ENDPOINT = "/workspace/lineplot/retrieve";
const CREATE_ENDPOINT = "/workspace/lineplot/create";
const RENAME_ENDPOINT = "/workspace/lineplot/rename";
const SET_DATA_ENDPOINT = "/workspace/lineplot/set-data";
const DELETE_ENDPOINT = "/workspace/lineplot/delete";

export const newLinePlotZ = linePlotZ.partial({ key: true }).transform((p) => ({
  ...p,
  data: JSON.stringify(p.data),
}));

export type NewLinePlot = z.input<typeof newLinePlotZ>;

const retrieveReqZ = z.object({ keys: keyZ.array() });
const createReqZ = z.object({ workspace: z.string(), linePlots: newLinePlotZ.array() });
const renameReqZ = z.object({ key: keyZ, name: z.string() });
const setDataReqZ = z.object({ key: keyZ, data: z.string() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const retrieveResZ = z.object({ linePlots: linePlotZ.array() });
const createResZ = z.object({ linePlots: linePlotZ.array() });
const emptyResZ = z.object({});

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(workspace: string, linePlot: NewLinePlot): Promise<LinePlot>;
  async create(workspace: string, linePlots: NewLinePlot[]): Promise<LinePlot[]>;
  async create(
    workspace: string,
    linePlots: NewLinePlot | NewLinePlot[],
  ): Promise<LinePlot | LinePlot[]> {
    const isMany = Array.isArray(linePlots);
    const normalized = toArray(linePlots);
    const res = await sendRequired(
      this.client,
      CREATE_ENDPOINT,
      { workspace, linePlots: normalized },
      createReqZ,
      createResZ,
    );
    return isMany ? res.linePlots : res.linePlots[0];
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

  async retrieve(key: Key): Promise<LinePlot>;
  async retrieve(keys: Key[]): Promise<LinePlot[]>;
  async retrieve(keys: Params): Promise<LinePlot | LinePlot[]> {
    const isMany = Array.isArray(keys);
    const normalized = toArray(keys);
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      { keys: normalized },
      retrieveReqZ,
      retrieveResZ,
    );
    return isMany ? res.linePlots : res.linePlots[0];
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
