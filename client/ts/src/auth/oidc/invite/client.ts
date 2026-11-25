// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { type Invite, inviteZ, type Key, keyZ } from "@/auth/oidc/invite/types.gen";
import { resource } from "@/resource";
import { user } from "@/user";

const createReqZ = z.object({ user: user.keyZ });
const createResZ = z.object({ token: z.string(), invite: inviteZ });
const retrieveReqZ = z.object({ subjects: resource.idZ.array().optional() });
const retrieveResZ = z.object({ invites: inviteZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const emptyResZ = z.object({});

const CREATE_ENDPOINT = "/auth/oidc/invite/create";
const RETRIEVE_ENDPOINT = "/auth/oidc/invite/retrieve";
const DELETE_ENDPOINT = "/auth/oidc/invite/delete";
/** The route that redeems an invite. The Core serves it without a session. */
export const REDEEM_ENDPOINT = "/auth/oidc/invite/redeem";
/** The route that redeems an invite with a password. It needs no session either. */
export const REDEEM_WITH_PASSWORD_ENDPOINT = "/auth/oidc/invite/redeem-password";

export interface CreateResult {
  /** The token that redeems the invite. It is the only copy of its secret. */
  token: string;
  /** The stored invite. */
  invite: Invite;
}

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  /**
   * Replaces the invite of a user.
   * @throws {ValidationError} when the Core does not accept the oidc method or the user
   * does not exist.
   * @throws {AccessDeniedError} when the caller may not invite another user.
   */
  async create(key: user.Key): Promise<CreateResult> {
    return await this.client.send(
      CREATE_ENDPOINT,
      { user: key },
      createReqZ,
      createResZ,
    );
  }

  /**
   * Lists live invites: the caller's own by default, or those of the given users when
   * the caller may retrieve them. A result never holds a token.
   * @throws {AccessDeniedError} when the caller may not retrieve another user's
   * invites.
   */
  async retrieve(subjects?: resource.ID[]): Promise<Invite[]> {
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
   * @throws {AccessDeniedError} when the caller may not revoke another user's invite.
   */
  async delete(...keys: Key[]): Promise<void> {
    await this.client.send(DELETE_ENDPOINT, { keys }, deleteReqZ, emptyResZ);
  }
}
