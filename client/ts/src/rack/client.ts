// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { array } from "@synnaxlabs/x";
import { z } from "zod";

import { ontology } from "@/ontology";
import {
  IDENTIFYING_FIELDS,
  RACK_FILTER_DESCRIPTOR,
  type RackRetrieveArg,
  type RackRetrieveByKey,
  type RackRetrieveByName,
  retrieveReqZ as rackRetrieveReqZ,
  retrieveResZ as rackRetrieveResZ,
} from "@/rack/filter.gen";
import {
  type Key,
  keyZ,
  type New,
  newZ,
  ontologyID,
  type Payload,
  payloadZ,
  type Status,
} from "@/rack/types.gen";
import { type task } from "@/task";
import { executeRetrieve, type RetrieveDescriptor } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_rack_set";
export const DELETE_CHANNEL_NAME = "sy_rack_delete";

export const rackZ = payloadZ;

const RETRIEVE: RetrieveDescriptor<Payload, "racks"> = {
  path: "/rack/retrieve",
  entityName: "Rack",
  reqZ: rackRetrieveReqZ,
  resZ: rackRetrieveResZ,
  itemsKey: "racks",
  filter: RACK_FILTER_DESCRIPTOR,
  identifyingFields: IDENTIFYING_FIELDS,
};

const createReqZ = z.object({ racks: newZ.array() });
const createResZ = z.object({ racks: payloadZ.array() });

const deleteReqZ = z.object({ keys: keyZ.array() });
const deleteResZ = z.object({});

export class Client {
  private readonly client: UnaryClient;
  private readonly tasks: task.Client;

  constructor(client: UnaryClient, taskClient: task.Client) {
    this.client = client;
    this.tasks = taskClient;
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      "/rack/delete",
      { keys: array.toArray(keys) },
      deleteReqZ,
      deleteResZ,
    );
  }

  async create(rack: New): Promise<Rack>;
  async create(racks: New[]): Promise<Rack[]>;
  async create(rack: New | New[]): Promise<Rack | Rack[]> {
    const isSingle = !Array.isArray(rack);
    const res = await sendRequired<typeof createReqZ, typeof createResZ>(
      this.client,
      "/rack/create",
      { racks: array.toArray(rack) },
      createReqZ,
      createResZ,
    );
    const sugared = this.sugar(res.racks);
    return isSingle ? sugared[0] : sugared;
  }

  async retrieve(args: RackRetrieveByKey): Promise<Rack>;
  async retrieve(args: RackRetrieveByName): Promise<Rack>;
  async retrieve(...args: [RackRetrieveArg, ...RackRetrieveArg[]]): Promise<Rack[]>;
  async retrieve(...args: RackRetrieveArg[]): Promise<Rack | Rack[]> {
    return executeRetrieve(RETRIEVE, this.client, args, (p) => this.sugar(p));
  }

  sugar(payload: Payload): Rack;
  sugar(payloads: Payload[]): Rack[];
  sugar(payloads: Payload | Payload[]): Rack | Rack[] {
    const isSingle = !Array.isArray(payloads);
    const sugared = array
      .toArray(payloads)
      .map(
        ({ key, name, status, integrations }) =>
          new Rack(key, name, this.tasks, status, integrations),
      );
    return isSingle ? sugared[0] : sugared;
  }
}

export class Rack {
  key: Key;
  name: string;
  status?: Status;
  integrations?: string[];
  private readonly tasks: task.Client;

  constructor(
    key: Key,
    name: string,
    taskClient: task.Client,
    status?: Status,
    integrations?: string[],
  ) {
    this.key = key;
    this.name = name;
    this.tasks = taskClient;
    this.status = status;
    this.integrations = integrations;
  }

  async listTasks(): Promise<task.Task[]> {
    return await this.tasks.retrieve({ rack: this.key });
  }

  async createTask(task: task.New): Promise<task.Task>;
  async createTask<Schemas extends task.Schemas = task.Schemas>(
    task: task.New<Schemas>,
    schemas: Schemas,
  ): Promise<task.Task<Schemas>>;

  async createTask<Schemas extends task.Schemas = task.Schemas>(
    task: task.New<Schemas>,
    schemas?: Schemas,
  ): Promise<task.Task<Schemas>> {
    task.key = (
      (BigInt(this.key) << 32n) +
      (BigInt(task.key ?? 0) & 0xffffffffn)
    ).toString();
    return await this.tasks.create(task, schemas as Required<Schemas>);
  }

  async deleteTask(task: task.Key): Promise<void> {
    await this.tasks.delete([task]);
  }

  get payload(): Payload {
    return {
      key: this.key,
      name: this.name,
      status: this.status,
      integrations: this.integrations,
    };
  }
}

export const statusKey = (key: Key): string => ontology.idToString(ontologyID(key));
