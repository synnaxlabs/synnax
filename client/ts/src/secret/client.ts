// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, primitive } from "@synnaxlabs/x";
import { z } from "zod";

import { ontology } from "@/ontology";
import { query } from "@/query";
import { rack } from "@/rack";
import {
  fileSecretZ,
  type Key,
  keyZ,
  ontologyID,
  type Secret,
  secretZ,
  storedSecretZ,
} from "@/secret/types.gen";

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
/** GrantKey is the key of a grant that lets a rack read secrets. */
export type GrantKey = z.infer<typeof grantKeyZ>;
const grantCreateReqZ = z.object({ rack: rack.keyZ, secrets: keyZ.array() });
export interface GrantParams extends z.input<typeof grantCreateReqZ> {}
const grantCreateResZ = z.object({ grant: grantKeyZ });
const grantDeleteReqZ = z.object({ grants: grantKeyZ.array() });

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  searchTerm: z.string().optional(),
  offset: z.number().optional(),
  limit: z.number().optional(),
});
const retrieveMultiParamsZ = retrieveRequestZ.or(query.keyListZ(keyZ));

export type RetrieveSingleParams = { key: Key };
export interface RetrieveMultipleParams extends z.input<typeof retrieveRequestZ> {}

interface RetrieveRequest extends z.infer<typeof retrieveRequestZ> {}

const retrieveResponseZ = z.object({ secrets: secretZ.array().default(() => []) });

const requestFilter = (req: RetrieveRequest): ((s: Secret) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  return (s) => keySet == null || keySet.has(s.key);
};

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
      fetch: async (keys) => await this.execRetrieve({ keys }),
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
        matches: (secret, req) => requestFilter(req)(secret),
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
   * Lets a rack read secrets that nothing on it uses yet, such as to test a connection
   * before saving the device that uses them. The grant ends when it is revoked or when
   * the client's session ends.
   * @returns the key of the grant to send to the rack with the command.
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

  /**
   * Revokes grants. Unknown keys are ignored.
   * @throws {AccessDeniedError} if the caller cannot create devices.
   */
  async revoke(grants: GrantKey | GrantKey[]): Promise<void> {
    await this.cfg.unary.send(
      "/secret/grant/delete",
      { grants: array.toArray(grants) },
      grantDeleteReqZ,
      emptyResZ,
    );
  }

  private async execRetrieve(params: RetrieveMultipleParams): Promise<Secret[]> {
    const res = await this.cfg.unary.send(
      "/secret/retrieve",
      params,
      retrieveRequestZ,
      retrieveResponseZ,
    );
    return res.secrets;
  }
}
