// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { UnaryClient } from "@synnaxlabs/freighter";
import { binary, type CrudeTimeStamp, TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
import { z } from "zod";

import { type APIKey, apiKeyZ, type Key, keyZ } from "@/auth/apikey/types.gen";
import { InvalidTokenError } from "@/errors";
import { resource } from "@/resource";

const TOKEN_PREFIX = "syk_";
/** A token holds the 16 bytes of the record key and a 32 byte secret. */
const TOKEN_LENGTH = 48;

/**
 * Returns the key of the API key record that a token belongs to.
 * @throws {InvalidTokenError} when the token is not an API key.
 */
export const keyOf = (token: string): Key => {
  if (!token.startsWith(TOKEN_PREFIX)) throw new InvalidTokenError("not an API key");
  let raw: Uint8Array;
  try {
    raw = binary.decodeBase64(token.slice(TOKEN_PREFIX.length), {
      alphabet: "base64url",
    });
  } catch {
    throw new InvalidTokenError("malformed API key");
  }
  if (raw.length !== TOKEN_LENGTH) throw new InvalidTokenError("malformed API key");
  return uuid.fromBytes(raw);
};

// A client clock that runs ahead of the Core's must not push a key past the limit.
const EXPIRY_MARGIN = TimeSpan.hours(1);

/**
 * @param maxLifetime - The longest lifetime of a new key, or null for no limit.
 * @returns The longest expiry to ask for: TimeStamp.MAX with no limit, else an hour
 * short of now plus the limit, or half the limit short when the limit is under two
 * hours.
 */
export const longestExpiry = (
  maxLifetime: TimeSpan | null,
  now: TimeStamp = TimeStamp.now(),
): TimeStamp => {
  if (maxLifetime == null) return TimeStamp.MAX;
  const margin = maxLifetime.lessThan(EXPIRY_MARGIN.mult(2))
    ? maxLifetime.div(2)
    : EXPIRY_MARGIN;
  return now.add(maxLifetime).sub(margin);
};

export interface New {
  /** The user or rack the key logs in as. Defaults to the caller. */
  subject?: resource.ID;
  /** A label that tells the subject's keys apart. */
  name: string;
  /** When the key stops logging in. Defaults to TimeStamp.MAX, which never expires. */
  expires?: CrudeTimeStamp;
}

const createReqZ = z.object({
  subject: resource.idZ.optional(),
  name: z.string(),
  expires: TimeStamp.z.default(TimeStamp.MAX),
});
const createResZ = z.object({ token: z.string(), key: apiKeyZ });
const retrieveReqZ = z.object({ subjects: resource.idZ.array().optional() });
const retrieveResZ = z.object({ keys: apiKeyZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const emptyResZ = z.object({});

const CREATE_ENDPOINT = "/auth/api-key/create";
const RETRIEVE_ENDPOINT = "/auth/api-key/retrieve";
const DELETE_ENDPOINT = "/auth/api-key/delete";
const ROTATE_ENDPOINT = "/auth/api-key/rotate";

export interface Rotated {
  /** The token to log in with. It is the only copy of the new key's secret. */
  token: string;
  /** The new key. */
  key: APIKey;
}

export class Client {
  private readonly client: UnaryClient;
  private readonly onRotate: (rotated: Rotated) => void;

  /**
   * @param onRotate - receives each key that replaces the key the client logged in
   * with.
   */
  constructor(client: UnaryClient, onRotate: (rotated: Rotated) => void = () => {}) {
    this.client = client;
    this.onRotate = onRotate;
  }

  /**
   * Creates an API key. The returned token is the only copy of the key's secret.
   * @throws {AccessDeniedError} when the caller may not create a key for the subject.
   * @throws {NotFoundError} when the subject does not exist.
   * @throws {PathError} when the expiry is not in the future.
   */
  async create(key: New): Promise<{ token: string; key: APIKey }> {
    return await this.client.send(CREATE_ENDPOINT, key, createReqZ, createResZ);
  }

  /**
   * Replaces the API key that the client logged in with by a new key for the same
   * subject and name. The new key lasts as long as the old key did, or an hour short
   * of the lifetime cap of the Core. The client's session stays open and later logins
   * use the new key. The old key cannot log in again.
   * @throws {ValidationError} when the client did not log in with an API key.
   * @throws {AuthError} when the key has expired.
   */
  async rotate(): Promise<Rotated> {
    const rotated = await this.client.send(ROTATE_ENDPOINT, {}, emptyResZ, createResZ);
    this.onRotate(rotated);
    return rotated;
  }

  /**
   * Lists API keys: the caller's own by default, or those of the given subjects when
   * the caller may retrieve them.
   * @throws {AccessDeniedError} when the caller may not retrieve another subject's
   * keys.
   */
  async retrieve(subjects?: resource.ID[]): Promise<APIKey[]> {
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
