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

import { type OIDCCredentials, oidcCredentialsZ } from "@/auth/types.gen";
import { resource } from "@/resource";
import { user } from "@/user";

export const linkKeyZ = z.uuid();
export type LinkKey = z.infer<typeof linkKeyZ>;

export const linkZ = z.object({
  /** key is the unique identifier for this link. */
  key: linkKeyZ,
  /** user is the key of the user the provider account logs in as. */
  user: user.keyZ,
  /**
   * issuer is the issuer URL of the provider that proved the account, its `iss` claim.
   * With account, it identifies the provider account.
   */
  issuer: z.string(),
  /** account is the provider's permanent ID for the account, its `sub` claim. */
  account: z.string(),
});
/** Link links an OpenID Connect provider account to a user. */
export interface Link extends z.infer<typeof linkZ> {}

const linkReqZ = z.object({ credentials: oidcCredentialsZ });
const linkResZ = z.object({ link: linkZ });
const retrieveReqZ = z.object({ subjects: resource.idZ.array().optional() });
const retrieveResZ = z.object({ links: linkZ.array() });
const unlinkReqZ = z.object({ keys: linkKeyZ.array() });
const emptyResZ = z.object({});

const LINK_ENDPOINT = "/auth/oidc/link";
const RETRIEVE_ENDPOINT = "/auth/oidc/retrieve";
const UNLINK_ENDPOINT = "/auth/oidc/unlink";

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  /**
   * Links the OpenID Connect provider account that the credentials prove to the
   * caller, so the account can log in as the caller.
   * @throws {InvalidCredentialsError} when the provider rejects the credentials.
   * @throws {ValidationError} when the account or the caller already has a link for
   * the provider.
   */
  async link(credentials: OIDCCredentials): Promise<Link> {
    const res = await this.client.send(
      LINK_ENDPOINT,
      { credentials },
      linkReqZ,
      linkResZ,
    );
    return res.link;
  }

  /**
   * Lists OpenID Connect provider links: the caller's own by default, or those of the
   * given users when the caller may retrieve them.
   * @throws {AccessDeniedError} when the caller may not retrieve another user's links.
   */
  async retrieve(subjects?: resource.ID[]): Promise<Link[]> {
    const res = await this.client.send(
      RETRIEVE_ENDPOINT,
      { subjects },
      retrieveReqZ,
      retrieveResZ,
    );
    return res.links;
  }

  /**
   * Removes OpenID Connect provider links and ends their sessions. Unknown keys are
   * ignored.
   * @throws {AccessDeniedError} when the caller may not remove another user's links.
   */
  async unlink(...keys: LinkKey[]): Promise<void> {
    await this.client.send(UNLINK_ENDPOINT, { keys }, unlinkReqZ, emptyResZ);
  }
}
