// Copyright 2025 Synnax Labs, Inc.
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
import { group } from "@/ontology/group";
import { checkForMultipleOrNoResults } from "@/util/retrieve";
import { nullableArrayZ } from "@/util/zod";
import {
  type Key,
  keyZ,
  type New,
  newZ,
  type Symbol,
  symbolZ,
} from "@/workspace/schematic/symbol/payload";

const RETRIEVE_ENDPOINT = "/workspace/schematic/symbol/retrieve";
const CREATE_ENDPOINT = "/workspace/schematic/symbol/create";
const RENAME_ENDPOINT = "/workspace/schematic/symbol/rename";
const DELETE_ENDPOINT = "/workspace/schematic/symbol/delete";
const RETRIEVE_GROUP_ENDPOINT = "/workspace/schematic/symbol/retrieve_group";

const createReqZ = z.object({ symbols: newZ.array(), parent: ontology.idZ });
const renameReqZ = z.object({ key: keyZ, name: z.string() });
const deleteReqZ = z.object({ keys: keyZ.array() });

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  searchTerm: z.string().optional(),
  offset: z.number().optional(),
  limit: z.number().optional(),
});

const singleRetrieveArgsZ = z
  .object({ key: keyZ })
  .transform(({ key }) => ({ keys: [key] }));

const retrieveArgsZ = z.union([singleRetrieveArgsZ, retrieveRequestZ]);

export type RetrieveArgs = z.input<typeof retrieveArgsZ>;
export type SingleRetrieveArgs = z.input<typeof singleRetrieveArgsZ>;
export type MultiRetrieveArgs = z.input<typeof retrieveRequestZ>;

const retrieveResZ = z.object({ symbols: nullableArrayZ(symbolZ) });
const createResZ = z.object({ symbols: symbolZ.array() });
const emptyResZ = z.object({});
const retrieveGroupReqZ = z.object({});
const retrieveGroupResZ = z.object({ group: group.groupZ });

export const SET_CHANNEL_NAME = "sy_schematic_symbol_set";
export const DELETE_CHANNEL_NAME = "sy_schematic_symbol_delete";

export interface CreateArgs extends New {
  parent: ontology.ID;
}

export interface CreateMultipleArgs {
  symbols: New[];
  parent: ontology.ID;
}

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(options: CreateArgs): Promise<Symbol>;
  async create(options: CreateMultipleArgs): Promise<Symbol[]>;
  async create(options: CreateArgs | CreateMultipleArgs): Promise<Symbol | Symbol[]> {
    const isMany = "symbols" in options;
    const symbols = isMany ? options.symbols : [options];
    const res = await sendRequired(
      this.client,
      CREATE_ENDPOINT,
      { symbols, parent: options.parent },
      createReqZ,
      createResZ,
    );
    return isMany ? res.symbols : res.symbols[0];
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

  async retrieve(args: SingleRetrieveArgs): Promise<Symbol>;
  async retrieve(args: MultiRetrieveArgs): Promise<Symbol[]>;
  async retrieve(args: RetrieveArgs): Promise<Symbol | Symbol[]> {
    const isSingle = "key" in args;
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      args,
      retrieveArgsZ,
      retrieveResZ,
    );
    checkForMultipleOrNoResults("Schematic Symbol", args, res.symbols, isSingle);
    return isSingle ? res.symbols[0] : res.symbols;
  }

  async delete(key: Key): Promise<void>;
  async delete(keys: Key[]): Promise<void>;
  async delete(keys: Key | Key[]): Promise<void> {
    await sendRequired(
      this.client,
      DELETE_ENDPOINT,
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }

  async retrieveGroup(): Promise<group.Payload> {
    const res = await sendRequired(
      this.client,
      RETRIEVE_GROUP_ENDPOINT,
      {},
      retrieveGroupReqZ,
      retrieveGroupResZ,
    );
    return res.group;
  }
}

export const ontologyID = (key: Key): ontology.ID => ({
  type: "schematic_symbol",
  key,
});
