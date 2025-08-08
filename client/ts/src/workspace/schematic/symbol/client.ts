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

import { type ontology } from "@/ontology";
import { checkForMultipleOrNoResults } from "@/util/retrieve";
import { nullableArrayZ } from "@/util/zod";
import {
  type Key,
  keyZ,
  type New,
  newZ,
  remoteZ,
  type Symbol,
} from "@/workspace/schematic/symbol/payload";

const RETRIEVE_ENDPOINT = "/workspace/schematic/symbol/retrieve";
const CREATE_ENDPOINT = "/workspace/schematic/symbol/create";
const RENAME_ENDPOINT = "/workspace/schematic/symbol/rename";
const DELETE_ENDPOINT = "/workspace/schematic/symbol/delete";

const createReqZ = z.object({ symbols: newZ.array() });
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

const retrieveResZ = z.object({ symbols: nullableArrayZ(remoteZ) });
const createResZ = z.object({ symbols: remoteZ.array() });
const emptyResZ = z.object({});

export const SET_CHANNEL_NAME = "sy_schematic_symbol_set";
export const DELETE_CHANNEL_NAME = "sy_schematic_symbol_delete";

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(symbol: New): Promise<Symbol>;
  async create(symbols: New[]): Promise<Symbol[]>;
  async create(symbols: New | New[]): Promise<Symbol | Symbol[]> {
    const isMany = Array.isArray(symbols);
    const res = await sendRequired(
      this.client,
      CREATE_ENDPOINT,
      { symbols: array.toArray(symbols) },
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
    checkForMultipleOrNoResults("Symbol", args, res.symbols, isSingle);
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
}

export const ontologyID = (key: Key): ontology.ID => ({ type: "schematic_symbol", key });