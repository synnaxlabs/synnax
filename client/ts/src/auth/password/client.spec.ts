// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { password } from "@/auth/password";
import {
  AuthError,
  InvalidCredentialsError,
  NotFoundError,
  ValidationError,
} from "@/index";
import {
  createEmptyRole,
  createTestAuthClient,
  createTestClient,
  createTestClientAs,
  createTestUser,
  TEST_CREDENTIALS,
} from "@/testutil";
import { user } from "@/user";

describe("password", () => {
  describe("set", () => {
    const createUser = async () => {
      const username = `set-password-${id.create()}`;
      const admin = createTestClient();
      const { key: role } = await createEmptyRole(admin);
      const u = await createTestUser(admin, {
        role,
        username,
        password: "old",
        name: "Set Password",
      });
      return { username, subject: user.resourceID(u.user.key) };
    };

    it("should let an owner set another user's password", async () => {
      const { username, subject } = await createUser();
      const owner = createTestAuthClient({ credentials: TEST_CREDENTIALS });
      await owner.passwords.set({ subject, password: "new" });
      await createTestAuthClient({
        credentials: { username, password: "new" },
      }).sessions.retrieve();
      await expect(
        createTestAuthClient({
          credentials: { username, password: "old" },
        }).sessions.retrieve(),
      ).rejects.toThrow(InvalidCredentialsError);
    });

    it("should switch to a new session when setting its own password", async () => {
      const { username } = await createUser();
      const successes: string[] = [];
      const self = createTestAuthClient(
        { credentials: { username, password: "old" } },
        { onSuccess: (key) => successes.push(key) },
      );
      await expect(self.passwords.set({ password: "new" })).rejects.toThrow();
      await self.sessions.retrieve();
      const old = self.session?.key;
      const before = successes.length;
      await self.passwords.set({ password: "new", currentPassword: "old" });
      expect(successes).toHaveLength(before + 1);
      expect(self.session?.key).toBeDefined();
      expect(successes.at(-1)).toEqual(self.session?.key);
      expect(self.session?.key).not.toEqual(old);
      const sessions = await self.sessions.retrieve();
      expect(sessions.map((s) => s.key)).not.toContain(old);
      await createTestAuthClient({
        credentials: { username, password: "new" },
      }).sessions.retrieve();
    });

    it("should end the replacement session when released", async () => {
      const { username } = await createUser();
      const self = createTestAuthClient({ credentials: { username, password: "old" } });
      await self.sessions.retrieve();
      await self.passwords.set({ password: "new", currentPassword: "old" });
      const replacement = self.session?.key;
      expect(replacement).toBeDefined();
      await self.release();
      const observer = createTestAuthClient({
        credentials: { username, password: "new" },
      });
      const sessions = await observer.sessions.retrieve();
      expect(sessions.map((s) => s.key)).not.toContain(replacement);
    });

    it("should keep new credentials set while a password change runs", async () => {
      const first = await createUser();
      const second = await createUser();
      const client = createTestAuthClient({
        credentials: { username: first.username, password: "old" },
      });
      await client.sessions.retrieve();
      const change = client.passwords.set({ password: "new", currentPassword: "old" });
      client.setCredentials({ username: second.username, password: "old" });
      await change;
      await client.sessions.retrieve();
      expect(client.subject?.key).toEqual(second.subject.key);
    });

    it("should keep working after a resumed client sets its own password", async () => {
      const { username } = await createUser();
      const opener = createTestAuthClient({
        credentials: { username, password: "old" },
      });
      await opener.sessions.retrieve();
      const token = opener.token;
      if (token == null) throw new Error("no token");
      const resumed = createTestAuthClient({ token });
      await resumed.passwords.set({ password: "new", currentPassword: "old" });
      expect(resumed.token).not.toEqual(token);
      await resumed.sessions.retrieve();
      await expect(opener.sessions.retrieve()).rejects.toThrow(InvalidCredentialsError);
    });

    it("should keep the owner's session when setting another's password", async () => {
      const { subject } = await createUser();
      const owner = createTestAuthClient({ credentials: TEST_CREDENTIALS });
      await owner.sessions.retrieve();
      const before = owner.session?.key;
      await owner.passwords.set({ subject, password: "new" });
      expect(owner.session?.key).toEqual(before);
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
      const { key: role } = await createEmptyRole(owner);
      const u = await owner.users.create({ name: "first password", role });
      const username = id.create();
      await owner.auth.passwords.create({
        subject: user.resourceID(u.key),
        username,
        password: "first",
      });
      const client = createTestAuthClient({
        credentials: { username, password: "first" },
      });
      await client.sessions.retrieve();
      expect(client.subject?.key).toEqual(u.key);
    });

    it("should refuse a username another user holds", async () => {
      const owner = createTestClient();
      const { key: role } = await createEmptyRole(owner);
      const { username } = await createTestUser(owner, { role });
      const u = await owner.users.create({ name: "repeated username", role });
      await expect(
        owner.auth.passwords.create({
          subject: user.resourceID(u.key),
          username,
          password: "test",
        }),
      ).rejects.toThrow(AuthError);
    });
  });

  describe("retrieve", () => {
    it("should retrieve the caller's own password by default", async () => {
      const owner = createTestClient();
      const { key: role } = await createEmptyRole(owner);
      const {
        user: u,
        username,
        password: secret,
      } = await createTestUser(owner, {
        role,
      });
      const client = createTestClientAs(username, secret);
      const passwords = await client.auth.passwords.retrieve();
      expect(passwords).toEqual([
        { key: expect.any(String), subject: user.resourceID(u.key), username },
      ]);
    });

    it("should retrieve a password by its username", async () => {
      const client = createTestClient();
      const { key: role } = await createEmptyRole(client);
      const { user: u, username } = await createTestUser(client, { role });
      const passwords = await client.auth.passwords.retrieve({ usernames: [username] });
      expect(passwords).toEqual([
        { key: expect.any(String), subject: user.resourceID(u.key), username },
      ]);
    });

    it("should retrieve a password by its subject", async () => {
      const client = createTestClient();
      const { key: role } = await createEmptyRole(client);
      const { user: u, username } = await createTestUser(client, { role });
      const subject = user.resourceID(u.key);
      const passwords = await client.auth.passwords.retrieve({ subjects: [subject] });
      expect(passwords).toEqual([{ key: expect.any(String), subject, username }]);
    });

    it("should ignore an unknown username", async () => {
      const client = createTestClient();
      const { key: role } = await createEmptyRole(client);
      const { user: u, username } = await createTestUser(client, { role });
      const passwords = await client.auth.passwords.retrieve({
        usernames: [username, id.create()],
      });
      expect(passwords).toEqual([
        { key: expect.any(String), subject: user.resourceID(u.key), username },
      ]);
    });
  });

  describe("dispatch", () => {
    it("should let an owner change another user's username", async () => {
      const owner = createTestClient();
      const { key: role } = await createEmptyRole(owner);
      const { user: u, password: secret } = await createTestUser(owner, { role });
      const [p] = await owner.auth.passwords.retrieve({
        subjects: [user.resourceID(u.key)],
      });
      const username = id.create();
      await owner.auth.passwords.dispatch(p.key, password.setUsername({ username }));
      const client = createTestAuthClient({
        credentials: { username, password: secret },
      });
      await client.sessions.retrieve();
      expect(client.subject?.key).toEqual(u.key);
    });

    it("should let a user change its own username and log in again", async () => {
      const owner = createTestClient();
      const { key: role } = await createEmptyRole(owner);
      const {
        user: u,
        username: old,
        password: secret,
      } = await createTestUser(owner, { role });
      // a self-rename needs update on the user, as any rename does
      await owner.access.policies.create({
        name: "self rename",
        role,
        objects: [user.resourceID(u.key)],
        actions: ["update"],
      });
      const client = createTestAuthClient({
        credentials: { username: old, password: secret },
      });
      const [p] = await client.passwords.retrieve();
      const username = id.create();
      await client.passwords.dispatch(p.key, password.setUsername({ username }));
      const key = client.session?.key;
      if (key == null) throw new Error("no session");
      await client.sessions.delete(key);
      const [renamed] = await client.passwords.retrieve();
      expect(renamed.username).toEqual(username);
      expect(client.session?.key).not.toEqual(key);
    });

    it("should refuse a username another user holds", async () => {
      const owner = createTestClient();
      const { key: role } = await createEmptyRole(owner);
      const { username } = await createTestUser(owner, { role });
      const { user: u } = await createTestUser(owner, { role });
      const [p] = await owner.auth.passwords.retrieve({
        subjects: [user.resourceID(u.key)],
      });
      await expect(
        owner.auth.passwords.dispatch(p.key, password.setUsername({ username })),
      ).rejects.toThrow(AuthError);
    });

    it("should refuse an unknown password", async () => {
      const owner = createTestClient();
      await expect(
        owner.auth.passwords.dispatch(
          uuid.create(),
          password.setUsername({ username: id.create() }),
        ),
      ).rejects.toThrow(NotFoundError);
    });
  });
});
