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

import { type Credentials } from "@/auth/types.gen";
import { ontology } from "@/ontology";

export interface CreateParams {
  /** The user to give the password. Defaults to the caller. */
  subject?: ontology.ID;
  /** The username to log in with. */
  username: string;
  /** The password. */
  password: string;
}

const createReqZ = z.object({
  subject: ontology.idZ.optional(),
  username: z.string(),
  password: z.string(),
});

export interface SetParams {
  /** The user whose password to set. Defaults to the caller. */
  subject?: ontology.ID;
  /** The new password. */
  password: string;
  /** The caller's current password. Required when the caller sets its own. */
  currentPassword?: string;
}

const setReqZ = z.object({
  subject: ontology.idZ.optional(),
  password: z.string(),
  currentPassword: z.string().optional(),
});

export const passwordZ = z.object({
  /** subject is the user the password logs in as. */
  subject: ontology.idZ,
  /** username is the name the password logs in with. */
  username: z.string(),
});
/** Password is a password as the API returns it: its subject and username. */
export interface Password extends z.infer<typeof passwordZ> {}

export interface RetrieveParams {
  /** The users whose passwords to list. */
  subjects?: ontology.ID[];
  /** The usernames of the passwords to list. */
  usernames?: string[];
}

const retrieveReqZ = z.object({
  subjects: ontology.idZ.array().optional(),
  usernames: z.string().array().optional(),
});
const retrieveResZ = z.object({ passwords: passwordZ.array().default(() => []) });

export interface RenameParams {
  /** The user whose password to rename. Defaults to the caller. */
  subject?: ontology.ID;
  /** The new username. */
  username: string;
}

const renameReqZ = z.object({ subject: ontology.idZ.optional(), username: z.string() });
const emptyResZ = z.object({});

const CREATE_ENDPOINT = "/auth/password/create";
const SET_ENDPOINT = "/auth/password/set";
const RETRIEVE_ENDPOINT = "/auth/password/retrieve";
const RENAME_ENDPOINT = "/auth/password/rename";

/**
 * Receives each change the Core accepts to a password, with the subject the caller
 * gave. An undefined subject means the caller's own password.
 */
export type ChangeHandler = (
  subject: ontology.ID | undefined,
  change: Partial<Credentials>,
) => void;

export class Client {
  private readonly client: UnaryClient;
  private readonly onChange: ChangeHandler;

  constructor(client: UnaryClient, onChange: ChangeHandler = () => {}) {
    this.client = client;
    this.onChange = onChange;
  }

  /**
   * Gives the caller, or another user when the caller may create that user, its first
   * password.
   * @throws {AuthError} when another user holds the username.
   * @throws {AccessDeniedError} when the caller may not create the user.
   * @throws {NotFoundError} when the user does not exist.
   * @throws {ValidationError} when the username or password is empty, or the user
   * already has a password.
   */
  async create(params: CreateParams): Promise<void> {
    await this.client.send(CREATE_ENDPOINT, params, createReqZ, emptyResZ);
  }

  /**
   * Sets the caller's password, or another user's when the caller may update that
   * user, and ends every other session the old password opened.
   * @throws {AuthError} when the caller sets its own password with a wrong current one.
   * @throws {AccessDeniedError} when the caller may not update the user.
   * @throws {ValidationError} when the password is empty, or the caller sets its own
   * without the current one.
   */
  async set(params: SetParams): Promise<void> {
    await this.client.send(SET_ENDPOINT, params, setReqZ, emptyResZ);
    this.onChange(params.subject, { password: params.password });
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
   * Changes the username of the caller's password, or of another user's when the caller
   * may update that user. Sessions stay open.
   * @throws {AuthError} when another user holds the username.
   * @throws {AccessDeniedError} when the caller may not update the user.
   * @throws {NotFoundError} when the user has no password.
   * @throws {ValidationError} when the username is empty.
   */
  async rename(params: RenameParams): Promise<void> {
    await this.client.send(RENAME_ENDPOINT, params, renameReqZ, emptyResZ);
    this.onChange(params.subject, { username: params.username });
  }
}
