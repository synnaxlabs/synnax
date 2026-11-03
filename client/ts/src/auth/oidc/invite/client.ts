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

import { type Invite, inviteZ, type Key, keyZ } from "@/auth/oidc/invite/types.gen";
import { ontology } from "@/ontology";

const createReqZ = z.object({ subject: ontology.idZ });
const createResZ = z.object({ token: z.string(), invite: inviteZ });
const retrieveReqZ = z.object({ subjects: ontology.idZ.array() });
const retrieveResZ = z.object({ invites: inviteZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const emptyResZ = z.object({});

const CREATE_ENDPOINT = "/auth/oidc/invite/create";
const RETRIEVE_ENDPOINT = "/auth/oidc/invite/retrieve";
const DELETE_ENDPOINT = "/auth/oidc/invite/delete";
/** The route that redeems an invite. The Core serves it without a session. */
export const REDEEM_ENDPOINT = "/auth/oidc/invite/redeem";

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  /**
   * Replaces the invite of a user. The returned token is the only copy of its secret.
   * @throws {ValidationError} when the Core does not accept the oidc method or the
   * subject is not a user.
   * @throws {AccessDeniedError} when the caller may not update the user.
   * @throws {NotFoundError} when the user does not exist.
   */
  async create(subject: ontology.ID): Promise<{ token: string; invite: Invite }> {
    return await this.client.send(CREATE_ENDPOINT, { subject }, createReqZ, createResZ);
  }

  /**
   * Lists the live invites of the given users. A result never holds a token.
   * @throws {ValidationError} when the Core does not accept the oidc method.
   * @throws {AccessDeniedError} when the caller may not retrieve a user.
   */
  async retrieve(...subjects: ontology.ID[]): Promise<Invite[]> {
    const res = await this.client.send(
      RETRIEVE_ENDPOINT,
      { subjects },
      retrieveReqZ,
      retrieveResZ,
    );
    return res.invites;
  }

  /**
   * Revokes invites. Unknown keys are ignored.
   * @throws {ValidationError} when the Core does not accept the oidc method.
   * @throws {AccessDeniedError} when the caller may not update an invited user.
   */
  async delete(...keys: Key[]): Promise<void> {
    await this.client.send(DELETE_ENDPOINT, { keys }, deleteReqZ, emptyResZ);
  }
}
