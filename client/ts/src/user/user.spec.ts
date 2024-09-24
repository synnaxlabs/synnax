// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { describe, expect, test } from "vitest";

import { AuthError, NotFoundError } from "@/errors";
import { newClient } from "@/setupspecs";
import { type user } from "@/user";

type SortType = { username: string };

const sort = (a: SortType, b: SortType) => a.username.localeCompare(b.username);

const client = newClient();

const userOne: user.NewUser = {
  username: id.id(),
  password: "test",
  firstName: "George",
  lastName: "Washington",
};

const userTwo: user.NewUser = { username: id.id(), password: "test" };

const userThree: user.NewUser = {
  username: id.id(),
  password: "test",
  firstName: "John",
  lastName: "Adams",
};

const userArray: user.NewUser[] = [
  { username: id.id(), password: "secondTest", firstName: "Steve" },
  { username: id.id(), password: "testArray" },
].sort(sort);

describe("User", () => {
  describe("Create", () => {
    describe("One", () => {
      test("with a name", async () => {
        const res = await client.user.create(userOne);
        expect(res.username).toEqual(userOne.username);
        expect(res.key).not.toEqual("");
        expect(res.firstName).toEqual(userOne.firstName);
        expect(res.lastName).toEqual(userOne.lastName);
        userOne.key = res.key;
      });
      test("with no name", async () => {
        const res = await client.user.create(userTwo);
        expect(res.username).toEqual(userTwo.username);
        expect(res.key).not.toEqual("");
        userTwo.key = res.key;
        expect(res.firstName).toEqual("");
        expect(res.lastName).toEqual("");
      });
      test("Repeated username", async () =>
        await expect(
          client.user.create({ username: userOne.username, password: "test" }),
        ).rejects.toThrow(AuthError));
    });
    describe("Many", () => {
      test("array empty", async () => {
        const res = await client.user.create([]);
        expect(res).toHaveLength(0);
      });
      test("array is one", async () => {
        const res = await client.user.create([userThree]);
        expect(res).toHaveLength(1);
        expect(res[0].username).toEqual(userThree.username);
        expect(res[0].key).not.toEqual("");
        userThree.key = res[0].key;
        expect(res[0].firstName).toEqual(userThree.firstName);
        expect(res[0].lastName).toEqual(userThree.lastName);
      });
      test("array not empty", async () => {
        const res = await client.user.create(userArray);
        expect(res).toHaveLength(2);
        userArray.forEach((u, i) => {
          expect(res[i].username).toEqual(u.username);
          expect(res[i].key).not.toEqual("");
          userArray[i].key = res[i].key;
          expect(res[i].firstName).toEqual(u.firstName ?? "");
          expect(res[i].lastName).toEqual(u.lastName ?? "");
        });
      });
      test("Repeated username", async () =>
        await expect(client.user.create([userOne, userTwo])).rejects.toThrow(
          AuthError,
        ));
    });
  });
  describe("Retrieve", () => {
    describe("by name", () => {
      describe("one", () => {
        test("found", async () => {
          const res = await client.user.retrieveByName(userOne.username);
          expect(res.username).toEqual(userOne.username);
          expect(res.key).toEqual(userOne.key);
          expect(res.firstName).toEqual(userOne.firstName);
          expect(res.lastName).toEqual(userOne.lastName);
        });
        test("not found", async () =>
          await expect(client.user.retrieveByName(id.id())).rejects.toThrow(
            NotFoundError,
          ));
      });
      describe("many", () => {
        test("found", async () => {
          const res = await client.user.retrieveByName(
            userArray.map((u) => u.username),
          );
          expect(res.sort(sort)).toHaveLength(2);
          res.forEach((u, i) => {
            expect(u.username).toEqual(userArray[i].username);
            expect(u.key).toEqual(userArray[i].key);
            expect(u.firstName).toEqual(userArray[i].firstName ?? "");
            expect(u.lastName).toEqual(userArray[i].lastName ?? "");
          });
        });
        test("not found", async () => {
          const res = await client.user.retrieveByName([id.id()]);
          expect(res).toEqual([]);
        });
        test("extra names getting deleted", async () => {
          const res = await client.user.retrieveByName([
            ...userArray.map((u) => u.username),
            id.id(),
          ]);
          expect(res.sort(sort)).toHaveLength(2);
          res.forEach((u, i) => {
            expect(u.username).toEqual(userArray[i].username);
            expect(u.key).toEqual(userArray[i].key);
            expect(u.firstName).toEqual(userArray[i].firstName ?? "");
            expect(u.lastName).toEqual(userArray[i].lastName ?? "");
          });
        });
        test("calling with no names", async () => {
          const res = await client.user.retrieveByName([]);
          const usernames = res.map((u) => u.username);
          expect(usernames).toContain(userOne.username);
          expect(usernames).toContain(userTwo.username);
          expect(usernames).toContain(userThree.username);
          userArray.forEach((u) => expect(usernames).toContain(u.username));
        });
      });
    });
    describe("by key", () => {
      describe("one", () => {
        test("found", async () => {
          const res = await client.user.retrieve(userOne.key as string);
          expect(res.username).toEqual(userOne.username);
          expect(res.key).toEqual(userOne.key);
          expect(res.firstName).toEqual(userOne.firstName);
          expect(res.lastName).toEqual(userOne.lastName);
        });
        test("not found", async () => {
          await expect(
            client.user.delete(userOne.key as string),
          ).resolves.toBeUndefined();
          await expect(client.user.retrieve(userOne.key as string)).rejects.toThrow(
            NotFoundError,
          );
          const u = await client.user.create(userOne);
          userOne.key = u.key;
        });
      });
      describe("many", () => {
        test("found", async () => {
          const res = await client.user.retrieve(userArray.map((u) => u.key as string));
          expect(res.sort(sort)).toHaveLength(2);
          res.forEach((u, i) => {
            expect(u.username).toEqual(userArray[i].username);
            expect(u.key).toEqual(userArray[i].key);
            expect(u.firstName).toEqual(userArray[i].firstName ?? "");
            expect(u.lastName).toEqual(userArray[i].lastName ?? "");
          });
        });
        test("not found", async () => {
          for (const u of userArray) {
            await expect(client.user.delete(u.key as string)).resolves.toBeUndefined();
          }
          await expect(
            client.user.retrieve(userArray.map((u) => u.key as string)),
          ).rejects.toThrow(NotFoundError);
          // cleanup
          const users = await client.user.create(userArray);
          users.forEach((u, i) => (userArray[i].key = u.key));
        });
        test("all", async () => {
          const res = await client.user.retrieve([]);
          const usernames = res.map((u) => u.username);
          expect(usernames).toContain(userOne.username);
          expect(usernames).toContain(userTwo.username);
          expect(usernames).toContain(userThree.username);
          userArray.forEach((u) => expect(usernames).toContain(u.username));
        });
      });
    });
  });
  describe("Change Username", () => {
    test("Successful", async () => {
      const newUsername = id.id();
      await expect(
        client.user.changeUsername(userOne.key as string, newUsername),
      ).resolves.toBeUndefined();
      const res = await client.user.retrieveByName(newUsername);
      expect(res.username).toEqual(newUsername);
      expect(res.key).not.toEqual("");
      expect(res.firstName).toEqual(userOne.firstName);
      expect(res.lastName).toEqual(userOne.lastName);
      userOne.username = newUsername;
    });
    test("Unsuccessful", async () =>
      await expect(
        client.user.changeUsername(userTwo.key as string, userOne.username),
      ).rejects.toThrow(AuthError));
    test(
      "Repeated usernames work",
      async () => {
        const oldUsername = id.id();
        console.log("old username", oldUsername);
        const user = await client.user.create({
          username: oldUsername,
          password: "test",
        });
        const newUsername = id.id();
        console.log("new username", newUsername);
        await client.user.changeUsername(user.key, newUsername);
        console.log("username changed");
        // below means username didn't actually change
        // const newUser = await client.user.create({username: newUsername, password: "test"});

        await expect(
          client.user.changeUsername(user.key, user.username),
        ).resolves.toBeUndefined();
      },
      1000 * 1000000,
    );
  });
  describe("Change Name", () => {
    test("Successful", async () => {
      await expect(
        client.user.rename(userOne.key as string, "Thomas", "Jefferson"),
      ).resolves.toBeUndefined();
      const res = await client.user.retrieve(userOne.key as string);
      expect(res.username).toEqual(userOne.username);
      expect(res.key).toEqual(userOne.key);
      expect(res.firstName).toEqual("Thomas");
      expect(res.lastName).toEqual("Jefferson");
      userOne.firstName = "Thomas";
      userOne.lastName = "Jefferson";
    });
    test("Only one name", async () => {
      await expect(
        client.user.rename(userOne.key as string, "James"),
      ).resolves.toBeUndefined();
      const res = await client.user.retrieve(userOne.key as string);
      expect(res.username).toEqual(userOne.username);
      expect(res.key).toEqual(userOne.key);
      expect(res.firstName).toEqual("James");
      expect(res.lastName).toEqual(userOne.lastName);
      userOne.firstName = "James";
    });
  });
  describe("Delete", () => {
    test("one that exists", async () => {
      await expect(client.user.delete(userOne.key as string)).resolves.toBeUndefined();
      await expect(client.user.retrieve(userOne.key as string)).rejects.toThrow(
        NotFoundError,
      );
    });
    test("many that exist", async () => {
      await expect(
        client.user.delete(userArray.map((u) => u.key as string)),
      ).resolves.toBeUndefined();
      await expect(
        client.user.retrieve(userArray.map((u) => u.key as string)),
      ).rejects.toThrow(NotFoundError);
    });
    test("one that doesn't exist", async () => {
      await expect(client.user.delete(userOne.key as string)).resolves.toBeUndefined();
    });
    test("many where some don't exist", async () => {
      await expect(
        client.user.delete([userOne.key as string, userTwo.key as string]),
      ).resolves.toBeUndefined();
      await expect(client.user.retrieve(userTwo.key as string)).rejects.toThrow(
        NotFoundError,
      );
    });
  });
});
