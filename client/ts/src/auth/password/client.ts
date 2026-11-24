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

import { type Action, dispatchReqZ } from "@/auth/password/actions.gen";
import { type Key, type Password, passwordZ } from "@/auth/password/types.gen";
import { type Credentials, type Session, sessionZ } from "@/auth/types.gen";
import { resource } from "@/resource";

export interface CreateParams {
  /** The user to give the password. Defaults to the caller. */
  subject?: resource.ID;
  /** The username to log in with. */
  username: string;
  /** The password. */
  password: string;
}

const createReqZ = z.object({
  subject: resource.idZ.optional(),
  username: z.string(),
  password: z.string(),
});

export interface SetParams {
  /** The user whose password to set. Defaults to the caller. */
  subject?: resource.ID;
  /** The new password. */
  password: string;
  /** The caller's current password. Required when the caller sets its own. */
  currentPassword?: string;
}

const setReqZ = z.object({
  subject: resource.idZ.optional(),
  password: z.string(),
  currentPassword: z.string().optional(),
  cookie: z.boolean().optional(),
});
const setResZ = z.object({ token: z.string(), session: sessionZ.optional() });

export interface RetrieveParams {
  /** The users whose passwords to list. */
  subjects?: resource.ID[];
  /** The usernames of the passwords to list. */
  usernames?: string[];
}

const retrieveReqZ = z.object({
  subjects: resource.idZ.array().optional(),
  usernames: z.string().array().optional(),
});
const retrieveResZ = z.object({ passwords: passwordZ.array().default(() => []) });

const emptyResZ = z.object({});

const CREATE_ENDPOINT = "/auth/password/create";
const SET_ENDPOINT = "/auth/password/set";
const RETRIEVE_ENDPOINT = "/auth/password/retrieve";
const DISPATCH_ENDPOINT = "/auth/password/dispatch";

/**
 * Receives each change the Core accepts to a password, with the subject the caller
 * gave. An undefined subject means the caller's own password.
 */
export type ChangeHandler = (
  subject: resource.ID | undefined,
  change: Partial<Credentials>,
) => void;

/** Receives the session the Core opens to replace the caller's own. */
export type SessionHandler = (token: string, session: Session) => void;

export interface Config {
  /** Whether the Core keeps the token in a cookie the client never sees. */
  cookie: boolean;
  /**
   * Returns a number that changes when the client's credentials change. A response to a
   * request sent under an older number is ignored.
   */
  generation: () => number;
  /** Returns the key of the credential the client's session logged in with. */
  credential: () => string | undefined;
  onChange: ChangeHandler;
  onSession: SessionHandler;
}

export class Client {
  private readonly client: UnaryClient;
  private readonly config: Config;

  constructor(client: UnaryClient, config: Config) {
    this.client = client;
    this.config = config;
  }

  /**
   * Gives the caller, or another user when the caller may create and update that user,
   * its first password.
   * @throws {AuthError} when another user holds the username.
   * @throws {AccessDeniedError} when the caller may not create and update the user.
   * @throws {NotFoundError} when the user does not exist.
   * @throws {ValidationError} when the username or password is empty, or the user
   * already has a password.
   */
  async create(params: CreateParams): Promise<void> {
    await this.client.send(CREATE_ENDPOINT, params, createReqZ, emptyResZ);
  }

  /**
   * Sets the caller's password, or another user's when the caller may update that
   * user, and ends every session the old password opened. When that ends the client's
   * own session, the client switches to the session the Core opens in its place.
   * @throws {AuthError} when the caller sets its own password with a wrong current one.
   * @throws {AccessDeniedError} when the caller may not update the user.
   * @throws {ValidationError} when the password is empty, or the caller sets its own
   * without the current one.
   */
  async set(params: SetParams): Promise<void> {
    const generation = this.config.generation();
    const res = await this.client.send(
      SET_ENDPOINT,
      { ...params, cookie: this.config.cookie || undefined },
      setReqZ,
      setResZ,
    );
    if (generation !== this.config.generation()) return;
    this.config.onChange(params.subject, { password: params.password });
    if (res.session != null) this.config.onSession(res.token, res.session);
  }

  /**
   * Lists passwords without their hashes: the caller's own by default, or those of the
   * given users or usernames when the caller may retrieve their users. Unknown users
   * and usernames are ignored.
   * @throws {AccessDeniedError} when the caller may not retrieve another user.
   */
  async retrieve(params: RetrieveParams = {}): Promise<Password[]> {
    const res = await this.client.send(
      RETRIEVE_ENDPOINT,
      params,
      retrieveReqZ,
      retrieveResZ,
    );
    return res.passwords;
  }

  /**
   * Applies the actions in order to the password with the given key. Sessions stay
   * open. When the client logged in with the password, it keeps its credentials in step
   * with the new username.
   * @throws {AuthError} when another user holds the username that the actions set.
   * @throws {AccessDeniedError} when the caller may not update the password's user.
   * @throws {NotFoundError} when no password has the key.
   * @throws {ValidationError} when an action sets an empty username.
   */
  async dispatch(key: Key, ...actions: Action[]): Promise<void> {
    const generation = this.config.generation();
    await this.client.send(
      DISPATCH_ENDPOINT,
      { key, dispatchKey: "", actions },
      dispatchReqZ,
      emptyResZ,
    );
    if (generation !== this.config.generation() || this.config.credential() !== key)
      return;
    for (const a of actions)
      this.config.onChange(undefined, { username: a.setUsername.username });
  }
}
