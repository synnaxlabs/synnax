// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Middleware, Unreachable } from "@synnaxlabs/freighter";
import { id, uuid } from "@synnaxlabs/x";
import { beforeAll, describe, expect, test, vi } from "vitest";
import { z } from "zod";

import { role } from "@/access/role";
import { AccessDeniedError, AuthError, NotFoundError, ValidationError } from "@/index";
import { query } from "@/query";
import {
  createEmptyRole,
  createTestClient,
  createTestClientAs,
  createTestClientWithPolicy,
  createTestUser,
  expectDeleted,
  expectLive,
  FAST_RETRY,
  holdResponses,
  spyOnSend,
  TEST_CREDENTIALS,
  waitForStreamLive,
} from "@/testutil";
import { user } from "@/user";

const client = createTestClient();

const empty = await createEmptyRole(client);

const retrieveRoot = async (): Promise<user.User> => {
  const [p] = await client.auth.passwords.retrieve({
    usernames: [TEST_CREDENTIALS.username],
  });
  return await client.users.retrieve(p.subject.key);
};

const createUser = async (r: role.Key = empty.key): Promise<user.User> =>
  await client.users.create({ name: "test", role: r });

const userOne: user.New = { name: "George Washington", role: empty.key };

const userTwo: user.New = { name: "John Hancock", role: empty.key };

const userThree: user.New = { name: "John Adams", role: empty.key };

const userArray: user.New[] = [
  { name: "Steve", role: empty.key },
  { name: "Martha", role: empty.key },
];

const createUsers = async (fixtures: user.New[]): Promise<user.User[]> =>
  await client.users.create(
    fixtures.map(({ key, name, role }) => ({ key, name, role })),
  );

/** Expects users to hold exactly the keys and names of the created users. */
const expectUsers = (users: user.User[], created: user.New[]) =>
  expect(users.map(({ key, name }) => ({ key, name }))).toEqual(
    expect.arrayContaining(created.map(({ key, name }) => ({ key, name }))),
  );

describe("User", () => {
  describe("Create", () => {
    describe("One", () => {
      test("with a name", async () => {
        const [res] = await createUsers([userOne]);
        expect(res.key).not.toEqual("");
        expect(res.name).toEqual(userOne.name);
        userOne.key = res.key;
      });
      test("with a different name", async () => {
        const [res] = await createUsers([userTwo]);
        expect(res.name).toEqual(userTwo.name);
        userTwo.key = res.key;
      });
    });
    describe("Many", () => {
      test("array empty", async () => {
        const res = await client.users.create([]);
        expect(res).toHaveLength(0);
      });
      test("array is one", async () => {
        const res = await createUsers([userThree]);
        expect(res).toHaveLength(1);
        expect(res[0].key).not.toEqual("");
        userThree.key = res[0].key;
        expect(res[0].name).toEqual(userThree.name);
      });
      test("array not empty", async () => {
        const res = await createUsers(userArray);
        expect(res).toHaveLength(2);
        userArray.forEach((u, i) => {
          expect(res[i].key).not.toEqual("");
          userArray[i].key = res[i].key;
          expect(res[i].name).toEqual(u.name);
        });
      });
      test("Repeated key", async () =>
        await expect(
          client.users.create({ key: userOne.key, name: "test", role: empty.key }),
        ).rejects.toThrow(ValidationError));
    });
    test("should not create a second user when it retries a create", async () => {
      let attempts = 0;
      // Fails the first create after the Core runs it, as a connection that drops
      // before the response headers arrive.
      const loseFirstResponse: Middleware = async (ctx, next) => {
        const out = await next(ctx);
        if (!ctx.target.endsWith("/user/create")) return out;
        attempts += 1;
        if (attempts === 1) throw new Unreachable({ message: "response lost" });
        return out;
      };
      const local = createTestClient({
        retry: FAST_RETRY,
        onRetry: () => {},
        middleware: [loseFirstResponse],
      });
      const send = spyOnSend(local);
      const name = id.create();
      await expect(local.users.create({ name, role: empty.key })).rejects.toThrow(
        ValidationError,
      );
      expect(attempts).toEqual(2);
      const [, req] =
        send.mock.calls.find(([target]) => target === "/user/create") ?? [];
      const { users } = z.object({ users: user.userZ.array() }).parse(req);
      const named = (await client.users.retrieve({})).filter((u) => u.name === name);
      expect(named.map((u) => u.key)).toEqual([users[0].key]);
    });
  });
  describe("Retrieve", () => {
    describe("by key", () => {
      describe("one", () => {
        test("found", async () => {
          const res = await client.users.retrieve(userOne.key as string);
          expect(res.key).toEqual(userOne.key);
          expect(res.name).toEqual(userOne.name);
        });
        test("not found", async () => {
          await expect(
            client.users.delete(userOne.key as string),
          ).resolves.toBeUndefined();
          await expect(client.users.retrieve(userOne.key as string)).rejects.toThrow(
            NotFoundError,
          );
          const [u] = await createUsers([userOne]);
          userOne.key = u.key;
        });
      });
      describe("many", () => {
        test("found", async () => {
          const res = await client.users.retrieve(
            userArray.map((u) => u.key as string),
          );
          expect(res).toHaveLength(2);
          expectUsers(res, userArray);
        });
        test("not found", async () => {
          for (const u of userArray)
            await expect(client.users.delete(u.key as string)).resolves.toBeUndefined();
          await expect(
            client.users.retrieve(userArray.map((u) => u.key as string)),
          ).rejects.toThrow(NotFoundError);
          const users = await createUsers(userArray);
          users.forEach((u, i) => (userArray[i].key = u.key));
        });
        test("all", async () => {
          const res = await client.users.retrieve({});
          expectUsers(res, [userOne, userTwo, userThree, ...userArray]);
        });
      });
    });
    describe("by where", () => {
      test("not and key", async () => {
        const keys = userArray.map((u) => u.key as string);
        const res = await client.users.retrieve({
          where: { not: { key: { eq: keys[0] } }, key: { in: keys } },
        });
        expect(res.map((u) => u.key)).toEqual([keys[1]]);
      });
      test("flat not and key", async () => {
        const keys = userArray.map((u) => u.key as string);
        const res = await client.users.retrieve({ not: { key: keys[0] }, key: keys });
        expect(res.map((u) => u.key)).toEqual([keys[1]]);
      });
      test("search term", async () => {
        const res = await client.users.retrieve({
          where: { key: { eq: userOne.key as string } },
          searchTerm: userOne.name,
        });
        expect(res.map((u) => u.key)).toEqual([userOne.key]);
      });
    });
  });
  describe("Change Name", () => {
    test("Successful", async () => {
      await expect(
        client.users.rename(userOne.key as string, "Thomas Jefferson"),
      ).resolves.toBeUndefined();
      const res = await client.users.retrieve(userOne.key as string);
      expect(res.key).toEqual(userOne.key);
      expect(res.name).toEqual("Thomas Jefferson");
      userOne.name = "Thomas Jefferson";
    });
    test("should keep a rename that streams in during a retrieve", async () => {
      const u = await createUser();
      const local = createTestClient();
      await local.connect();
      await waitForStreamLive(local.connection);
      await local.users.retrieve(u.key);
      const cachedName = () => {
        const cached = local.users.watch(u.key).getCached();
        return query.isLive(cached) ? cached.name : undefined;
      };
      const { responded, release } = holdResponses(local, "/user/retrieve");
      const fetching = local.users.retrieve(u.key);
      await responded;
      const name = `renamed-${id.create()}`;
      await client.users.rename(u.key, name);
      await expect.poll(cachedName).toBe(name);
      release();
      await fetching;
      expect(cachedName()).toBe(name);
    });
  });
  describe("Change Username", () => {
    test("should let an owner change another user's username", async () => {
      const { user: u, password } = await createTestUser(client, { role: empty.key });
      const username = id.create();
      await client.users.changeUsername(u.key, username);
      const renamed = createTestClientAs(username, password);
      const [p] = await renamed.auth.passwords.retrieve();
      expect(p.subject).toEqual(user.resourceID(u.key));
    });
    test("should keep logging in after a user changes its own username", async () => {
      const r = await createEmptyRole(client);
      const {
        user: u,
        username: old,
        password,
      } = await createTestUser(client, { role: r.key });
      // a self-rename needs update on the user, as any rename does
      await client.access.policies.create({
        name: "self rename",
        role: r.key,
        objects: [user.resourceID(u.key)],
        actions: ["update"],
      });
      const self = createTestClientAs(old, password);
      const username = id.create();
      await self.users.changeUsername(u.key, username);
      const [p] = await client.auth.passwords.retrieve({
        subjects: [user.resourceID(u.key)],
      });
      expect(p.username).toEqual(username);
      const key = self.auth.session?.key;
      if (key == null) throw new Error("no session");
      await self.auth.sessions.delete(key);
      await self.auth.sessions.retrieve();
      expect(self.auth.session?.key).not.toEqual(key);
    });
    test("should refuse a username another user holds", async () => {
      const { username } = await createTestUser(client, { role: empty.key });
      const { user: u } = await createTestUser(client, { role: empty.key });
      await expect(client.users.changeUsername(u.key, username)).rejects.toThrow(
        AuthError,
      );
    });
    test("should refuse a user with no password", async () => {
      const u = await createUser();
      await expect(client.users.changeUsername(u.key, id.create())).rejects.toThrow(
        NotFoundError,
      );
    });
  });
  describe("Role", () => {
    test("create a user with a role", async () => {
      const r = await createEmptyRole(client);
      const u = await createUser(r.key);
      expect(u.role).toEqual(r.key);
    });
    test("retrieve the users that hold a role", async () => {
      const r = await createEmptyRole(client);
      const u = await createUser(r.key);
      await createUser();
      const held = await client.users.retrieve({ roles: [r.key] });
      expect(held.map(({ key }) => key)).toEqual([u.key]);
    });
  });
  describe("Set Role", () => {
    test("replace the role of users", async () => {
      const [from, to] = await Promise.all([
        createEmptyRole(client),
        createEmptyRole(client),
      ]);
      const a = await createUser(from.key);
      const b = await createUser();
      await client.users.setRole([a.key, b.key], to.key);
      expect(expectLive(client.users.watch(a.key).getCached()).role).toEqual(to.key);
      const fresh = createTestClient();
      const stored = await fresh.users.retrieve([a.key, b.key]);
      expect(stored.map((u) => u.role)).toEqual([to.key, to.key]);
    });
    test("refuse to change the role of the root user", async () => {
      const r = await createEmptyRole(client);
      const root = await retrieveRoot();
      await expect(client.users.setRole(root.key, r.key)).rejects.toThrow(
        new ValidationError(`cannot change the role of root user ${root.name}`),
      );
      expect(expectLive(client.users.watch(root.key).getCached()).role).toEqual(
        root.role,
      );
      const fresh = createTestClient();
      expect((await fresh.users.retrieve(root.key)).role).toEqual(root.role);
    });
    test("change no role when one user cannot take it", async () => {
      const [from, to] = await Promise.all([
        createEmptyRole(client),
        createEmptyRole(client),
      ]);
      const u = await createUser(from.key);
      const root = await retrieveRoot();
      await expect(client.users.setRole([u.key, root.key], to.key)).rejects.toThrow(
        `cannot change the role of root user ${root.name}`,
      );
      const fresh = createTestClient();
      expect((await fresh.users.retrieve(u.key)).role).toEqual(from.key);
    });
    test("refuse a user that does not exist", async () => {
      const r = await createEmptyRole(client);
      await expect(client.users.setRole(uuid.create(), r.key)).rejects.toThrow(
        NotFoundError,
      );
    });
    test("refuse a role that does not exist", async () => {
      const u = await createUser();
      const missing = uuid.create();
      await expect(client.users.setRole(u.key, missing)).rejects.toThrow(
        `role: role ${missing} does not exist`,
      );
    });
    test("deny a subject that cannot update the role the user holds", async () => {
      const [from, to] = await Promise.all([
        createEmptyRole(client),
        createEmptyRole(client),
      ]);
      const u = await createUser(from.key);
      const subject = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [user.TYPE_RESOURCE_ID, role.resourceID(to.key)],
        actions: ["update"],
      });
      await expect(subject.users.setRole(u.key, to.key)).rejects.toSatisfy(
        AccessDeniedError.matches,
      );
      const fresh = createTestClient();
      expect((await fresh.users.retrieve(u.key)).role).toEqual(from.key);
    });
    test("allow a subject that can update the users and both roles", async () => {
      const [from, to] = await Promise.all([
        createEmptyRole(client),
        createEmptyRole(client),
      ]);
      const u = await createUser(from.key);
      const subject = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [
          user.TYPE_RESOURCE_ID,
          role.resourceID(from.key),
          role.resourceID(to.key),
        ],
        actions: ["update"],
      });
      await subject.users.setRole(u.key, to.key);
      const fresh = createTestClient();
      expect((await fresh.users.retrieve(u.key)).role).toEqual(to.key);
    });
  });
  describe("Dispatch", () => {
    test("should apply the action to the user", async () => {
      const u = await createUser();
      const name = `dispatched-${id.create()}`;
      await client.users.dispatch(u.key, user.set({ path: ["name"], value: name }));
      expect((await createTestClient().users.retrieve(u.key)).name).toEqual(name);
    });
  });
  describe("Delete", () => {
    test("one that exists", async () => {
      await expect(client.users.delete(userOne.key as string)).resolves.toBeUndefined();
      await expect(client.users.retrieve(userOne.key as string)).rejects.toThrow(
        NotFoundError,
      );
    });
    test("many that exist", async () => {
      await expect(
        client.users.delete(userArray.map((u) => u.key as string)),
      ).resolves.toBeUndefined();
      await expect(
        client.users.retrieve({ key: userArray.map((u) => u.key as string) }),
      ).resolves.toEqual([]);
    });
    test("one that doesn't exist", async () => {
      await expect(client.users.delete(userOne.key as string)).resolves.toBeUndefined();
    });
    test("many where some don't exist", async () => {
      await expect(
        client.users.delete([userOne.key as string, userTwo.key as string]),
      ).resolves.toBeUndefined();
      await expect(client.users.retrieve(userTwo.key as string)).rejects.toThrow(
        NotFoundError,
      );
    });
  });
});

// A second client with its own cache: its writes reach the first client only through
// the cluster's change streams, never through a shared cache.
const remote = createTestClient();

describe("cached reads", () => {
  // Changes made while the stream is still opening are lost, so open it up front.
  beforeAll(async () => await client.connect());

  test("should deliver a remote rename to a subscribed user", async () => {
    const u = await createUser();
    const handler = vi.fn();
    const off = client.users.watch(u.key).onChange(handler);
    try {
      await client.users.retrieve(u.key);
      await remote.users.rename(u.key, "Abigail Adams");
      await expect
        .poll(() => {
          const cached = client.users.watch(u.key).getCached();
          return query.isLive(cached) && cached.name === "Abigail Adams";
        })
        .toBe(true);
      expect(handler).toHaveBeenCalledWith(
        expect.objectContaining({ name: "Abigail Adams", key: u.key }),
      );
    } finally {
      off();
    }
  });

  test("should reduce a root_user set locally before the Core rejects it", async () => {
    const u = await createUser();
    await client.users.retrieve(u.key);
    // The client applies the set to the held user before it sends. A client that could
    // not apply it would throw a plain error, not the Core's validation error.
    await expect(
      client.users.dispatch(u.key, user.set({ path: ["root_user"], value: true })),
    ).rejects.toThrow(ValidationError);
    expect(expectLive(client.users.watch(u.key).getCached()).rootUser).toBe(false);
  });

  test("should deliver a remote role change to a subscribed user", async () => {
    const r = await createEmptyRole(client);
    const u = await createUser();
    await client.users.retrieve(u.key);
    await remote.users.setRole(u.key, r.key);
    await expect
      .poll(() => expectLive(client.users.watch(u.key).getCached()).role)
      .toEqual(r.key);
  });

  test("should deliver a remote delete as a deleted result", async () => {
    const u = await createUser();
    const results: Array<query.Cached<user.User> | undefined> = [];
    const off = client.users.watch(u.key).onChange((res) => results.push(res));
    try {
      await client.users.retrieve(u.key);
      await remote.users.delete(u.key);
      await expect
        .poll(() => query.Deleted.matches(client.users.watch(u.key).getCached()))
        .toBe(true);
      expect(expectDeleted(results.at(-1)).corpse.key).toEqual(u.key);
    } finally {
      off();
    }
  });
});
