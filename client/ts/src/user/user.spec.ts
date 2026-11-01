// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { beforeAll, describe, expect, test, vi } from "vitest";

import { AuthError, NotFoundError, ValidationError } from "@/index";
import { query } from "@/query";
import { createTestClient, expectDeleted } from "@/testutil";
import { type user } from "@/user";

const client = createTestClient();

const userOne: user.New = {
  username: id.create(),
  password: "test",
  name: "George Washington",
};

const userTwo: user.New = { username: id.create(), password: "test" };

const userThree: user.New = {
  username: id.create(),
  password: "test",
  name: "John Adams",
};

const userArray: user.New[] = [
  { username: id.create(), password: "secondTest", name: "Steve" },
  { username: id.create(), password: "testArray" },
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
          client.users.create({ username: userOne.username, password: "test" }),
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
            { username: userOne.username, password: "test" },
            { username: userTwo.username, password: "test" },
          ]),
        ).rejects.toThrow(AuthError));
      test("Repeated key", async () =>
        await expect(
          client.users.create({
            key: userOne.key,
            username: id.create(),
            password: "test",
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
        test("calling with no names", async () => {
          const res = await client.users.retrieve({ usernames: [] });
          expectUsers(res, [userOne, userTwo, userThree, ...userArray]);
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
          const res = await client.users.retrieve({
            keys: userArray.map((u) => u.key as string),
          });
          expect(res).toHaveLength(2);
          expectUsers(res, userArray);
        });
        test("not found", async () => {
          for (const u of userArray)
            await expect(client.users.delete(u.key as string)).resolves.toBeUndefined();
          await expect(
            client.users.retrieve({ keys: userArray.map((u) => u.key as string) }),
          ).rejects.toThrow(NotFoundError);
          const users = await client.users.create(userArray);
          users.forEach((u, i) => (userArray[i].key = u.key));
        });
        test("all", async () => {
          const res = await client.users.retrieve({ keys: [] });
          expectUsers(res, [userOne, userTwo, userThree, ...userArray]);
        });
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
      });
      const newUsername = id.create();
      await client.users.changeUsername(user.key, newUsername);
      await expect(
        client.users.create({ username: newUsername, password: "test" }),
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
        client.users.retrieve({ keys: userArray.map((u) => u.key as string) }),
      ).rejects.toThrow(NotFoundError);
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

const createUser = async (): Promise<user.User> =>
  await client.users.create({ username: id.create(), password: "test" });

describe("cached reads", () => {
  // Changes made while the stream is still opening are lost, so open it up front.
  beforeAll(async () => await client.connect());

  test("should deliver a remote rename to a subscribed user", async () => {
    const u = await createUser();
    const handler = vi.fn();
    const off = client.users.onChange(u.key, handler);
    try {
      await client.users.retrieve(u.key);
      await remote.users.rename(u.key, "Abigail Adams");
      await expect
        .poll(() => {
          const cached = client.users.getCached(u.key);
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

  test("should deliver a remote delete as a deleted result", async () => {
    const u = await createUser();
    const results: Array<query.Cached<user.User> | undefined> = [];
    const off = client.users.onChange(u.key, (res) => results.push(res));
    try {
      await client.users.retrieve(u.key);
      await remote.users.delete(u.key);
      await expect
        .poll(() => query.Deleted.matches(client.users.getCached(u.key)))
        .toBe(true);
      expect(expectDeleted(results.at(-1)).corpse.key).toEqual(u.key);
    } finally {
      off();
    }
  });
});
