// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { type Session, type SessionKey, sessionKeyZ, sessionZ } from "@/auth/types.gen";
import { ontology } from "@/ontology";

const retrieveReqZ = z.object({ subjects: ontology.idZ.array().optional() });
const retrieveResZ = z.object({ sessions: sessionZ.array() });
export const deleteReqZ = z.object({ keys: sessionKeyZ.array() });
const deleteResZ = z.object({});

export const RETRIEVE_ENDPOINT = "/auth/session/retrieve";
export const DELETE_ENDPOINT = "/auth/session/delete";

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  /**
   * Lists sessions: the caller's own by default, or those of the given subjects when
   * the caller may retrieve them.
   * @throws {AccessDeniedError} when the caller may not retrieve another subject's
   * sessions.
   * @throws {AuthError} when the login before the request fails.
   */
  async retrieve(subjects?: ontology.ID[]): Promise<Session[]> {
    const res = await this.client.send(
      RETRIEVE_ENDPOINT,
      { subjects },
      retrieveReqZ,
      retrieveResZ,
    );
    return res.sessions;
  }

  /**
   * Ends the given sessions. Unknown keys are ignored.
   * @throws {AccessDeniedError} when the caller may not end another subject's sessions.
   * @throws {AuthError} when the login before the request fails.
   */
  async delete(...keys: SessionKey[]): Promise<void> {
    await this.client.send(DELETE_ENDPOINT, { keys }, deleteReqZ, deleteResZ);
  }
}
