// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { TimeStamp, uuid } from "@synnaxlabs/x";
import { z } from "zod";

import { InvalidTokenError } from "@/errors";
import { ontology } from "@/ontology";

export const keyZ = z.uuid();
export type Key = z.infer<typeof keyZ>;

export const apiKeyZ = z.object({
  /** key is the unique identifier for this API key. */
  key: keyZ,
  /** subject is the user or rack the key logs in as. */
  subject: ontology.idZ,
  /** name is a label that tells the subject's keys apart. */
  name: z.string(),
  /** created is when the key was created. */
  created: TimeStamp.z,
});
/** APIKey is an API key as the API returns it: the stored record without its hash. */
export interface APIKey extends z.infer<typeof apiKeyZ> {}

const TOKEN_PREFIX = "syk_";
/** A token holds the 16 bytes of the record key and a 32 byte secret. */
const TOKEN_LENGTH = 48;

/**
 * Returns the key of the API key record that a token belongs to.
 * @throws {InvalidTokenError} when the token is not an API key.
 */
export const keyOf = (token: string): Key => {
  if (!token.startsWith(TOKEN_PREFIX)) throw new InvalidTokenError("not an API key");
  const body = token
    .slice(TOKEN_PREFIX.length)
    .replaceAll("-", "+")
    .replaceAll("_", "/");
  let raw: Uint8Array;
  try {
    raw = Uint8Array.from(atob(body), (c) => c.charCodeAt(0));
  } catch {
    throw new InvalidTokenError("malformed API key");
  }
  if (raw.length !== TOKEN_LENGTH) throw new InvalidTokenError("malformed API key");
  return uuid.parse(raw);
};

export interface New {
  /** The user or rack the key logs in as. Defaults to the caller. */
  subject?: ontology.ID;
  /** A label that tells the subject's keys apart. */
  name: string;
}

const createReqZ = z.object({ subject: ontology.idZ.optional(), name: z.string() });
const createResZ = z.object({ token: z.string(), key: apiKeyZ });
const retrieveReqZ = z.object({ subjects: ontology.idZ.array().optional() });
const retrieveResZ = z.object({ keys: apiKeyZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const emptyResZ = z.object({});

const CREATE_ENDPOINT = "/auth/api-key/create";
const RETRIEVE_ENDPOINT = "/auth/api-key/retrieve";
const DELETE_ENDPOINT = "/auth/api-key/delete";

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  /**
   * Creates an API key. The returned token is the only copy of the key's secret.
   * @throws {AccessDeniedError} when the caller may not create a key for the subject.
   * @throws {NotFoundError} when the subject does not exist.
   */
  async create(key: New): Promise<{ token: string; key: APIKey }> {
    return await this.client.send(CREATE_ENDPOINT, key, createReqZ, createResZ);
  }

  /**
   * Lists API keys: the caller's own by default, or those of the given subjects when
   * the caller may retrieve them.
   * @throws {AccessDeniedError} when the caller may not retrieve another subject's
   * keys.
   */
  async retrieve(subjects?: ontology.ID[]): Promise<APIKey[]> {
    const res = await this.client.send(
      RETRIEVE_ENDPOINT,
      { subjects },
      retrieveReqZ,
      retrieveResZ,
    );
    return res.keys;
  }

  /**
   * Deletes API keys and ends their sessions. Unknown keys are ignored.
   * @throws {AccessDeniedError} when the caller may not delete another subject's keys.
   */
  async delete(...keys: Key[]): Promise<void> {
    await this.client.send(DELETE_ENDPOINT, { keys }, deleteReqZ, emptyResZ);
  }
}
