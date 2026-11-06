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

import { ontology } from "@/ontology";
import { query } from "@/query";
import {
  fileSecretZ,
  type Key,
  keyZ,
  ontologyID,
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
  type Secret,
  secretZ,
  storedSecretZ,
} from "@/secret/types.gen";
import { matches, retrieveInputZ, serverComputed } from "@/secret/where.gen";

export const SET_CHANNEL_NAME = "sy_secret_set";
export const DELETE_CHANNEL_NAME = "sy_secret_delete";

const newStoredZ = storedSecretZ.omit({ updated: true }).extend({
  /** value is the hidden value. No client can read it back. */
  value: z.string(),
});
const newFileZ = fileSecretZ.omit({ updated: true });

/**
 * A secret to create, or to replace when its key exists. A stored secret carries its
 * value, which the Core encrypts and never returns.
 */
export const newZ = z.discriminatedUnion("source", [newStoredZ, newFileZ]);
export type New = z.input<typeof newZ>;

const createReqZ = z.object({ secrets: newZ.array() });
const createResZ = z.object({ secrets: secretZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const renameReqZ = z.object({ key: keyZ, name: z.string() });
const emptyResZ = z.object({});

const grantKeyZ = z.uuid();
/** GrantKey is the key of a grant that lets a subject read secrets once. */
export type GrantKey = z.infer<typeof grantKeyZ>;
const grantCreateReqZ = z.object({ subject: ontology.idZ, secrets: keyZ.array() });
export interface GrantParams extends z.input<typeof grantCreateReqZ> {}
const grantCreateResZ = z.object({ grant: grantKeyZ });

const retrieveMultiParamsZ = z.union([
  query.keysRequestZ(keyZ),
  retrieveRequestZ,
  retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape),
]);

export type RetrieveSingleParams = { key: Key };
export type RetrieveMultipleParams = query.RequestInput<typeof retrieveMultiParamsZ>;

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  ontology: ontology.Client;
}

export class Client extends query.Retriever<typeof retrieveMultiParamsZ, Key, Secret> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Secret>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const store = cache.createTable<Key, Secret>({
      name: "secrets",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, secretZ),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
    super(cache, {
      name: "secret",
      table: store,
      request: {
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (secret, req) => matches(secret, req.where),
        serverComputed: (req) => serverComputed(req.where),
      },
    });
    this.cfg = cfg;
    this.store = store;
  }

  /**
   * Creates secrets, or replaces the secrets whose keys exist.
   * @returns the saved secrets, without their values.
   * @throws {PathError} if a secret is not valid, such as a stored secret without a
   * value.
   * @throws {AccessDeniedError} if the caller cannot create secrets, or cannot update a
   * secret that the call replaces.
   */
  async create(secret: New): Promise<Secret>;
  async create(secrets: New[]): Promise<Secret[]>;
  async create(secrets: New | New[]): Promise<Secret | Secret[]> {
    const isMany = Array.isArray(secrets);
    const res = await this.cfg.unary.send(
      "/secret/create",
      { secrets: array.toArray(secrets) },
      createReqZ,
      createResZ,
    );
    this.store.set(res.secrets);
    return isMany ? res.secrets : res.secrets[0];
  }

  /**
   * Renames a secret.
   * @throws {PathError} if the name is empty.
   * @throws {NotFoundError} if no secret has the key.
   * @throws {AccessDeniedError} if the caller cannot update the secret.
   */
  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    const rename = () => [
      query.partialUpdate(this.store, key, { name }),
      this.cfg.ontology.cache.renameResource(ontologyID(key), name),
    ];
    await query.optimistic({
      rollbacks: rename(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/secret/rename",
          { key, name },
          renameReqZ,
          emptyResZ,
        ),
    });
    rename();
  }

  /**
   * Deletes secrets. Unknown keys are ignored.
   * @throws {ValidationError} if a resource uses one of the secrets.
   * @throws {AccessDeniedError} if the caller cannot delete one of the secrets.
   */
  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const ids = ontologyID(keysArr);
    const drop = () => [
      this.cfg.ontology.cache.deleteRelationships(ids),
      this.store.delete(keysArr),
      this.cfg.ontology.cache.resources.delete(ontology.idToString(ids)),
    ];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/secret/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    });
    drop();
  }

  /**
   * Lets a subject read secrets once, within a minute, such as for a rack to test a
   * connection before the device that uses them is saved. Only racks can be subjects.
   * @returns the key of the grant to send to the subject with the command.
   * @throws {AccessDeniedError} if the caller cannot create devices.
   * @throws {NotFoundError} if a secret does not exist.
   */
  async grant(params: GrantParams): Promise<GrantKey> {
    const res = await this.cfg.unary.send(
      "/secret/grant/create",
      params,
      grantCreateReqZ,
      grantCreateResZ,
    );
    return res.grant;
  }

  private async execRetrieve(req: RetrieveRequest): Promise<Secret[]> {
    const res = await this.cfg.unary.send(
      "/secret/retrieve",
      req,
      retrieveRequestZ,
      retrieveResponseZ,
    );
    return res.secrets;
  }
}
