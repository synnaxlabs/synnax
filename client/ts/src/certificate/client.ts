// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, filter } from "@synnaxlabs/x";
import { z } from "zod";

import { actions } from "@/actions";
import { createOf, reduceAll } from "@/certificate/actions";
import {
  type Action,
  dispatchReqZ,
  dispatchResZ,
  scopedActionZ,
  set,
} from "@/certificate/actions.gen";
import {
  type Certificate,
  certificateZ,
  fileCertificateZ,
  type Key,
  keyZ,
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
  storedCertificateZ,
} from "@/certificate/types.gen";
import {
  compareCursors,
  cursor,
  LIST_FIELDS,
  matches,
  retrieveInputZ,
  serverComputed,
  shortcutZ,
} from "@/certificate/where.gen";
import { query } from "@/query";

export const ACTION_CHANNEL_NAME = "sy_certificate_action_set";
export const DELETE_CHANNEL_NAME = "sy_certificate_delete";

const newStoredZ = storedCertificateZ.omit({ details: true, privateKey: true }).extend({
  /**
   * privateKeyPem is the private key as PEM. The Core checks that it matches the
   * certificate and saves it as a secret.
   */
  privateKeyPem: z.string().optional(),
});

/**
 * A certificate to create, or to replace when its key exists. The Core reads the
 * details of a stored certificate from its PEM.
 */
export const newZ = z.discriminatedUnion("source", [newStoredZ, fileCertificateZ]);
export type New = z.input<typeof newZ>;

const setReqZ = z.object({ certificates: newZ.array() });
const setResZ = z.object({ certificates: certificateZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const emptyResZ = z.object({});

const retrieveMultiParamsZ = filter.requestInputZ(
  retrieveRequestZ,
  retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape),
);

export type RetrieveSingleParams = { key: Key };
export type RetrieveMultipleParams = query.RequestInput<typeof retrieveMultiParamsZ>;

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
}

export class Client extends query.Retriever<
  typeof retrieveMultiParamsZ,
  Key,
  Certificate
> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Certificate>;
  private readonly dispatcher: actions.Controller<Key, Certificate, Action>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const store = cache.createTable<Key, Certificate>({
      name: "certificates",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    // Every write to a certificate reaches the client as actions.
    const dispatcher = new actions.Controller<Key, Certificate, Action>({
      store,
      listed: (action) => actions.changesField(action, LIST_FIELDS),
      onError: cache.onError,
      onEpoch: (callback) => cache.onEpoch(callback),
      onStreamDown: (callback) => cache.onStreamDown(callback),
      send: actions.sender(
        cfg.unary,
        "/certificate/dispatch",
        dispatchReqZ,
        dispatchResZ,
      ),
      reduce: reduceAll,
      createOf,
    });
    cache.listen(dispatcher.listener(ACTION_CHANNEL_NAME, scopedActionZ));
    super(cache, {
      name: "certificate",
      table: store,
      shortcut: shortcutZ,
      request: {
        list: { cursor, compare: compareCursors },
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (certificate, req) => matches(certificate, req.where),
        serverComputed: (req) => serverComputed(req.where),
      },
    });
    this.cfg = cfg;
    this.store = store;
    this.dispatcher = dispatcher;
  }

  /**
   * Creates certificates, or replaces the certificates whose keys exist.
   * @throws {PathError} if a certificate has no name, a PEM does not parse, or a
   * private key does not match its certificate.
   * @throws {AccessDeniedError} if the caller may not create certificates, create
   * secrets for private keys, or update a certificate it replaces and the secret of its
   * private key.
   */
  async set(certificate: New): Promise<Certificate>;
  async set(certificates: New[]): Promise<Certificate[]>;
  async set(certificates: New | New[]): Promise<Certificate | Certificate[]> {
    const isMany = Array.isArray(certificates);
    const res = await this.cfg.unary.send(
      "/certificate/set",
      { certificates: array.toArray(certificates) },
      setReqZ,
      setResZ,
    );
    this.store.set(res.certificates);
    return isMany ? res.certificates : res.certificates[0];
  }

  /**
   * Renames a certificate.
   * @throws {PathError} if the name is empty.
   * @throws {NotFoundError} if no certificate has the key.
   * @throws {AccessDeniedError} if the caller may not update the certificate, or the
   * secret of its private key, which takes the new name.
   */
  async rename(key: Key, name: string): Promise<void> {
    await this.dispatch(key, set({ path: ["name"], value: name }));
  }

  /**
   * Applies the actions to the cached certificate until the Core echoes them, and sends
   * them to the Core to apply in one transaction. Path actions edit only the name, and
   * a new name also renames the secret of the private key.
   * @throws {Error} if the cached certificate cannot hold an action, before any send.
   * @throws {ValidationError} if the certificate cannot hold an action.
   * @throws {NotFoundError} if the certificate does not exist.
   * @throws {AccessDeniedError} if the caller may not update the certificate, or the
   * secret of its private key.
   */
  async dispatch(key: Key, actions: Action | Action[]): Promise<void> {
    await this.dispatcher.send(key, array.toArray(actions));
  }

  /**
   * Deletes certificates and the secrets that hold their private keys.
   * @throws {AccessDeniedError} if the caller may not delete a certificate or the
   * secret of its private key.
   */
  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [this.store.delete(keysArr)];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/certificate/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    });
    drop();
  }

  private async execRetrieve(req: RetrieveRequest): Promise<Certificate[]> {
    const res = await this.cfg.unary.send(
      "/certificate/retrieve",
      req,
      retrieveRequestZ,
      retrieveResponseZ,
    );
    return res.certificates;
  }
}
