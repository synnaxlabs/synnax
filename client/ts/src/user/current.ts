// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, observe } from "@synnaxlabs/x";

import { type connection } from "@/connection";
import { UnexpectedError } from "@/errors";
import { query } from "@/query";
import { type Key, type User } from "@/user/types.gen";

export interface CurrentConfig {
  users: query.Retrieves<Key, User>;
  table: query.Table<Key, User>;
  /** Carries the user after each login, and undefined when the credentials change. */
  identity: observe.Observable<User | undefined>;
  connection: connection.Client;
}

/**
 * The cached read surface for the signed-in user. It follows each login, so a
 * subscription made before the first login, or held while someone else signs in,
 * tracks whoever is signed in.
 */
export class Current {
  private readonly users: query.Retrieves<Key, User>;
  private readonly connection: connection.Client;
  private readonly switches = new observe.Observer<void>();
  private key?: Key;

  constructor({ users, table, identity, connection }: CurrentConfig) {
    this.users = users;
    this.connection = connection;
    identity.onChange((user) => {
      if (user != null) table.set(user);
      if (user?.key === this.key) return;
      this.key = user?.key;
      this.switches.notify();
    });
  }

  /**
   * Retrieves the signed-in user, waiting for the connection when no user is signed in.
   * @throws {AuthError} if the Core rejects the credentials.
   * @throws {DisconnectedError} if the Core cannot be reached.
   */
  async retrieve(): Promise<User> {
    if (this.key == null) await this.connection.connect();
    const { key } = this;
    if (key == null) throw new UnexpectedError("no user is signed in after connecting");
    // A login returns the user without a retrieve grant, so its record comes first.
    const cached = this.users.getCached(key);
    if (query.isLive(cached)) return cached;
    return await this.users.retrieve(key);
  }

  /**
   * Calls the handler each time the signed-in user changes: an edit to the user, or a
   * login as a different user. Returns a destructor that unsubscribes.
   */
  onChange(handler: query.ChangeHandler<User>): destructor.Destructor {
    let release = this.follow(handler);
    const stop = this.switches.onChange(() => {
      release?.();
      release = this.follow(handler);
      handler(this.getCached());
    });
    return () => {
      stop();
      release?.();
    };
  }

  /** @returns The cached signed-in user, or undefined when no user is signed in. */
  getCached(): query.Cached<User> | undefined {
    if (this.key == null) return undefined;
    return this.users.getCached(this.key);
  }

  private follow(
    handler: query.ChangeHandler<User>,
  ): destructor.Destructor | undefined {
    if (this.key == null) return undefined;
    return this.users.onChange(this.key, handler);
  }
}
