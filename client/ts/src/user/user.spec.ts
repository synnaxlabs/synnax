// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, uuid } from "@synnaxlabs/x";
import { beforeAll, describe, expect, test, vi } from "vitest";

import { role } from "@/access/role";
import { AccessDeniedError, AuthError, NotFoundError, ValidationError } from "@/index";
import { query } from "@/query";
import {
  createEmptyRole,
  createTestClient,
  createTestClientWithPolicy,
  expectDeleted,
  expectLive,
  holdResponses,
  TEST_CREDENTIALS,
  waitForStreamLive,
} from "@/testutil";
import { user } from "@/user";

const client = createTestClient();

const empty = await createEmptyRole(client);

const createUser = async (r: role.Key = empty.key): Promise<user.User> =>
  await client.users.create({ username: id.create(), password: "test", role: r });

const userOne: user.New = {
  username: id.create(),
  password: "test",
  name: "George Washington",
  role: empty.key,
};

const userTwo: user.New = { username: id.create(), password: "test", role: empty.key };

const userThree: user.New = {
  username: id.create(),
  password: "test",
  name: "John Adams",
  role: empty.key,
};

const userArray: user.New[] = [
  { username: id.create(), password: "secondTest", name: "Steve", role: empty.key },
  { username: id.create(), password: "testArray", role: empty.key },
];

/** Expects users to hold exactly the keys and names of the created users. */
const expectUsers = (users: user.User[], created: user.New[]) =>
  expect(users.map(({ key, name }) => ({ key, name }))).toEqual(
    expect.arrayContaining(
      created.map((u) => ({ key: u.key, name: u.name ?? u.username })),
    ),
  );

describe("User", () => {
  describe("Create", () => {
    describe("One", () => {
      test("with a name", async () => {
        const res = await client.users.create(userOne);
        expect(res.key).not.toEqual("");
        expect(res.name).toEqual(userOne.name);
        userOne.key = res.key;
      });
      test("with no name", async () => {
        const res = await client.users.create(userTwo);
        expect(res.key).not.toEqual("");
        userTwo.key = res.key;
        expect(res.name).toEqual(userTwo.username);
      });
      test("Repeated username", async () =>
        await expect(
          client.users.create({
            username: userOne.username,
            password: "test",
            role: empty.key,
          }),
        ).rejects.toThrow(AuthError));
    });
    describe("Many", () => {
      test("array empty", async () => {
        const res = await client.users.create([]);
        expect(res).toHaveLength(0);
      });
      test("array is one", async () => {
        const res = await client.users.create([userThree]);
        expect(res).toHaveLength(1);
        expect(res[0].key).not.toEqual("");
        userThree.key = res[0].key;
        expect(res[0].name).toEqual(userThree.name);
      });
      test("array not empty", async () => {
        const res = await client.users.create(userArray);
        expect(res).toHaveLength(2);
        userArray.forEach((u, i) => {
          expect(res[i].key).not.toEqual("");
          userArray[i].key = res[i].key;
          expect(res[i].name).toEqual(u.name ?? u.username);
        });
      });
      test("Repeated username", async () =>
        await expect(
          client.users.create([
            { username: userOne.username, password: "test", role: empty.key },
            { username: userTwo.username, password: "test", role: empty.key },
          ]),
        ).rejects.toThrow(AuthError));
      test("Repeated key", async () =>
        await expect(
          client.users.create({
            key: userOne.key,
            username: id.create(),
            password: "test",
            role: empty.key,
          }),
        ).rejects.toThrow(ValidationError));
    });
  });
  describe("Retrieve", () => {
    describe("by name", () => {
      describe("one", () => {
        test("found", async () => {
          const res = await client.users.retrieve({ username: userOne.username });
          expect(res.key).toEqual(userOne.key);
          expect(res.name).toEqual(userOne.name);
        });
        test("not found", async () =>
          await expect(
            client.users.retrieve({ username: id.create() }),
          ).rejects.toThrow(NotFoundError));
      });
      describe("many", () => {
        test("found", async () => {
          const res = await client.users.retrieve({
            usernames: userArray.map((u) => u.username),
          });
          expect(res).toHaveLength(2);
          expectUsers(res, userArray);
        });
        test("not found", async () => {
          const res = await client.users.retrieve({ usernames: [id.create()] });
          expect(res).toEqual([]);
        });
        test("extra names getting deleted", async () => {
          const res = await client.users.retrieve({
            usernames: [...userArray.map((u) => u.username), id.create()],
          });
          expect(res).toHaveLength(2);
          expectUsers(res, userArray);
        });
      });
    });
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
          const u = await client.users.create(userOne);
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
          const users = await client.users.create(userArray);
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
  describe("Change Username", () => {
    test("Successful", async () => {
      const newUsername = id.create();
      await expect(
        client.users.changeUsername(userOne.key as string, newUsername),
      ).resolves.toBeUndefined();
      const res = await client.users.retrieve({ username: newUsername });
      expect(res.key).toEqual(userOne.key);
      expect(res.name).toEqual(userOne.name);
      userOne.username = newUsername;
    });
    test("Unsuccessful", async () =>
      await expect(
        client.users.changeUsername(userTwo.key as string, userOne.username),
      ).rejects.toThrow(AuthError));
    test("Repeated usernames fail", async () => {
      const oldUsername = id.create();
      const user = await client.users.create({
        username: oldUsername,
        password: "test",
        role: empty.key,
      });
      const newUsername = id.create();
      await client.users.changeUsername(user.key, newUsername);
      await expect(
        client.users.create({
          username: newUsername,
          password: "test",
          role: empty.key,
        }),
      ).rejects.toThrow(AuthError);
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
      const root = await client.users.retrieve({
        username: TEST_CREDENTIALS.username,
      });
      await expect(client.users.setRole(root.key, r.key)).rejects.toThrow(
        new ValidationError(
          `cannot change the role of root user ${root.name}: validation error`,
        ),
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
      const root = await client.users.retrieve({
        username: TEST_CREDENTIALS.username,
      });
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
        `role: role ${missing} does not exist: validation error`,
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
