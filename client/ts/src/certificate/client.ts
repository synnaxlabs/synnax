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

import {
  baseZ,
  type Certificate,
  certificateZ,
  fileCertificateZ,
  type Key,
  keyZ,
  ontologyID,
} from "@/certificate/types.gen";
import { ontology } from "@/ontology";
import { query } from "@/query";

export const SET_CHANNEL_NAME = "sy_certificate_set";
export const DELETE_CHANNEL_NAME = "sy_certificate_delete";

const newStoredZ = baseZ.extend({
  source: z.literal("stored"),
  /** pem is the certificate chain as PEM, leaf first. */
  pem: z.string(),
  /**
   * privateKey is the private key as PEM. The Core checks that it matches the
   * certificate and saves it as a secret.
   */
  privateKey: z.string().optional(),
});

/**
 * A certificate to create, or to replace when its key exists. The Core reads the
 * details of a stored certificate from its PEM.
 */
export const newZ = z.discriminatedUnion("source", [newStoredZ, fileCertificateZ]);
export type New = z.input<typeof newZ>;

const createReqZ = z.object({ certificates: newZ.array() });
const createResZ = z.object({ certificates: certificateZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const renameReqZ = z.object({ key: keyZ, name: z.string() });
const emptyResZ = z.object({});

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

const retrieveResponseZ = z.object({
  certificates: certificateZ.array().default(() => []),
});

const requestFilter = (req: RetrieveRequest): ((c: Certificate) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  return (c) => keySet == null || keySet.has(c.key);
};

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  ontology: ontology.Client;
}

export class Client extends query.Retriever<
  typeof retrieveMultiParamsZ,
  Key,
  Certificate
> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Certificate>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const store = cache.createTable<Key, Certificate>({
      name: "certificates",
      fetch: async (keys) => await this.execRetrieve({ keys }),
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, certificateZ),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
    super(cache, {
      name: "certificate",
      table: store,
      request: {
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (certificate, req) => requestFilter(req)(certificate),
      },
    });
    this.cfg = cfg;
    this.store = store;
  }

  /**
   * Creates certificates, or replaces the certificates whose keys exist.
   * @throws if a PEM does not parse, or a private key does not match its certificate.
   */
  async create(certificate: New): Promise<Certificate>;
  async create(certificates: New[]): Promise<Certificate[]>;
  async create(certificates: New | New[]): Promise<Certificate | Certificate[]> {
    const isMany = Array.isArray(certificates);
    const res = await this.cfg.unary.send(
      "/certificate/create",
      { certificates: array.toArray(certificates) },
      createReqZ,
      createResZ,
    );
    this.store.set(res.certificates);
    return isMany ? res.certificates : res.certificates[0];
  }

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
          "/certificate/rename",
          { key, name },
          renameReqZ,
          emptyResZ,
        ),
    });
    rename();
  }

  /**
   * Deletes certificates.
   * @throws if a device uses one of the certificates.
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
          "/certificate/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    });
    drop();
  }

  private async execRetrieve(params: RetrieveMultipleParams): Promise<Certificate[]> {
    const res = await this.cfg.unary.send(
      "/certificate/retrieve",
      params,
      retrieveRequestZ,
      retrieveResponseZ,
    );
    return res.certificates;
  }
}
