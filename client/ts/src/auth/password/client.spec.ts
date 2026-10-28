// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { AuthError, NotFoundError, ValidationError } from "@/errors";
import { createTestAuthClient, createTestClient, createTestUser } from "@/testutil";
import { user } from "@/user";

describe("password", () => {
  describe("set", () => {
    const createUser = async () => {
      const username = `set-password-${id.create()}`;
      const u = await createTestUser(createTestClient(), { username, password: "old" });
      return { username, subject: user.ontologyID(u.user.key) };
    };

    it("should let an owner set another user's password", async () => {
      const { username, subject } = await createUser();
      const owner = createTestClient();
      await owner.auth.passwords.set({ subject, password: "new" });
      await createTestAuthClient({ username, password: "new" }).sessions.retrieve();
      await expect(
        createTestAuthClient({ username, password: "old" }).sessions.retrieve(),
      ).rejects.toThrow(AuthError);
    });

    it("should set the caller's own password given the current one", async () => {
      const { username } = await createUser();
      const self = createTestAuthClient({ username, password: "old" });
      await expect(self.passwords.set({ password: "new" })).rejects.toThrow();
      await self.sessions.retrieve();
      await self.passwords.set({
        subject: self.subject,
        password: "new",
        currentPassword: "old",
      });
      const key = self.session?.key;
      if (key == null) throw new Error("no session");
      await self.sessions.delete(key);
      await self.sessions.retrieve();
      expect(self.session?.key).not.toEqual(key);
      await createTestAuthClient({ username, password: "new" }).sessions.retrieve();
    });
  });

  describe("create", () => {
    it("should refuse a user that already has a password", async () => {
      const client = createTestClient();
      await expect(
        client.auth.passwords.create({ username: id.create(), password: "p" }),
      ).rejects.toThrow(ValidationError);
    });

    it("should let an owner give a new user its first password", async () => {
      const owner = createTestClient();
      const u = await owner.users.create({ name: "first password" });
      const username = id.create();
      await owner.auth.passwords.create({
        subject: user.ontologyID(u.key),
        username,
        password: "first",
      });
      const client = createTestAuthClient({ username, password: "first" });
      await client.sessions.retrieve();
      expect(client.subject?.key).toEqual(u.key);
    });

    it("should refuse a username another user holds", async () => {
      const owner = createTestClient();
      const { username } = await createTestUser(owner);
      const u = await owner.users.create({ name: "repeated username" });
      await expect(
        owner.auth.passwords.create({
          subject: user.ontologyID(u.key),
          username,
          password: "test",
        }),
      ).rejects.toThrow(AuthError);
    });
  });

  describe("retrieve", () => {
    it("should retrieve the caller's own password by default", async () => {
      const owner = createTestClient();
      const { user: u, username, password } = await createTestUser(owner);
      const client = createTestAuthClient({ username, password });
      const passwords = await client.passwords.retrieve();
      expect(passwords).toEqual([{ subject: user.ontologyID(u.key), username }]);
    });

    it("should retrieve a password by its username", async () => {
      const client = createTestClient();
      const { user: u, username } = await createTestUser(client);
      const passwords = await client.auth.passwords.retrieve({ usernames: [username] });
      expect(passwords).toEqual([{ subject: user.ontologyID(u.key), username }]);
    });

    it("should retrieve a password by its subject", async () => {
      const client = createTestClient();
      const { user: u, username } = await createTestUser(client);
      const subject = user.ontologyID(u.key);
      const passwords = await client.auth.passwords.retrieve({ subjects: [subject] });
      expect(passwords).toEqual([{ subject, username }]);
    });

    it("should ignore an unknown username", async () => {
      const client = createTestClient();
      const { user: u, username } = await createTestUser(client);
      const passwords = await client.auth.passwords.retrieve({
        usernames: [username, id.create()],
      });
      expect(passwords).toEqual([{ subject: user.ontologyID(u.key), username }]);
    });
  });

  describe("rename", () => {
    it("should let an owner rename another user's password", async () => {
      const owner = createTestClient();
      const { user: u, password } = await createTestUser(owner);
      const username = id.create();
      await owner.auth.passwords.rename({ subject: user.ontologyID(u.key), username });
      const client = createTestAuthClient({ username, password });
      await client.sessions.retrieve();
      expect(client.subject?.key).toEqual(u.key);
    });

    it("should let a user rename its own password and log in again", async () => {
      const owner = createTestClient();
      const { user: u, username: old, password } = await createTestUser(owner);
      const client = createTestAuthClient({ username: old, password });
      const username = id.create();
      await client.passwords.rename({ username });
      const [p] = await owner.auth.passwords.retrieve({
        subjects: [user.ontologyID(u.key)],
      });
      expect(p.username).toEqual(username);
      const key = client.session?.key;
      if (key == null) throw new Error("no session");
      await client.sessions.delete(key);
      await client.sessions.retrieve();
      expect(client.session?.key).not.toEqual(key);
    });

    it("should refuse a username another user holds", async () => {
      const owner = createTestClient();
      const { username } = await createTestUser(owner);
      const { user: u } = await createTestUser(owner);
      await expect(
        owner.auth.passwords.rename({ subject: user.ontologyID(u.key), username }),
      ).rejects.toThrow(AuthError);
    });

    it("should refuse a user with no password", async () => {
      const owner = createTestClient();
      const u = await owner.users.create({ name: "no password" });
      await expect(
        owner.auth.passwords.rename({
          subject: user.ontologyID(u.key),
          username: id.create(),
        }),
      ).rejects.toThrow(NotFoundError);
    });
  });
});
