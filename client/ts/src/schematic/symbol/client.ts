// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array } from "@synnaxlabs/x";
import { z } from "zod";

import { group } from "@/group";
import { ontology } from "@/ontology";
import {
  type Key,
  keyZ,
  type New,
  type Symbol,
  symbolZ,
} from "@/schematic/symbol/types.gen";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

const createReqZ = z.object({ symbols: symbolZ.array(), parent: ontology.idZ });
const renameReqZ = z.object({ key: keyZ, name: z.string() });
const deleteReqZ = z.object({ keys: keyZ.array() });

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  searchTerm: z.string().optional(),
});

const singleRetrieveParamsZ = z
  .object({ key: keyZ })
  .transform(({ key }) => ({ keys: [key] }));

const retrieveParamsZ = z.union([singleRetrieveParamsZ, retrieveRequestZ]);

export type RetrieveParams = z.input<typeof retrieveParamsZ>;
export type RetrieveSingleParams = z.input<typeof singleRetrieveParamsZ>;
export type RetrieveMultipleParams = z.input<typeof retrieveRequestZ>;

const retrieveResZ = z.object({ symbols: symbolZ.array().default(() => []) });
const createResZ = z.object({ symbols: symbolZ.array() });
const emptyResZ = z.object({});
const retrieveGroupReqZ = z.object({});
const retrieveGroupResZ = z.object({ group: group.groupZ });

export const SET_CHANNEL_NAME = "sy_schematic_symbol_set";
export const DELETE_CHANNEL_NAME = "sy_schematic_symbol_delete";

export interface CreateParams extends New {
  parent: ontology.ID;
}

export interface CreateMultipleParams {
  symbols: New[];
  parent: ontology.ID;
}

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(options: CreateParams): Promise<Symbol>;
  async create(options: CreateMultipleParams): Promise<Symbol[]>;
  async create(
    options: CreateParams | CreateMultipleParams,
  ): Promise<Symbol | Symbol[]> {
    const isMany = "symbols" in options;
    const symbols = isMany ? options.symbols : [options];
    const res = await this.client.send(
      "/schematic/symbol/create",
      { symbols, parent: options.parent },
      createReqZ,
      createResZ,
    );
    return isMany ? res.symbols : res.symbols[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    await this.client.send(
      "/schematic/symbol/rename",
      { key, name },
      renameReqZ,
      emptyResZ,
    );
  }

  async retrieve(args: RetrieveSingleParams): Promise<Symbol>;
  async retrieve(args: RetrieveMultipleParams): Promise<Symbol[]>;
  async retrieve(args: RetrieveParams): Promise<Symbol | Symbol[]> {
    const isSingle = "key" in args;
    const res = await this.client.send(
      "/schematic/symbol/retrieve",
      args,
      retrieveParamsZ,
      retrieveResZ,
    );
    checkForMultipleOrNoResults("Schematic Symbol", args, res.symbols, isSingle);
    return isSingle ? res.symbols[0] : res.symbols;
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await this.client.send(
      "/schematic/symbol/delete",
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }

  async retrieveGroup(): Promise<group.Group> {
    const res = await this.client.send(
      "/schematic/symbol/retrieve-group",
      {},
      retrieveGroupReqZ,
      retrieveGroupResZ,
    );
    return res.group;
  }
}
