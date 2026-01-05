// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Interface for objects that can be discriminated by a string identifier.
 * Classes or objects implementing this interface must have a discriminator property
 * that uniquely identifies their type.
 *
 * @example
 * ```typescript
 * class User implements Discriminated {
 *   discriminator = "user";
 *   constructor(public name: string) {}
 * }
 * ```
 */
export interface Discriminated {
  discriminator: string;
}

/**
 * Creates a type guard function that checks if a value is an instance of a class
 * using a discriminator property. It is up to the caller to ensure that the
 * discriminator property is unique, consistent across all instances of the class,
 * and will not appear in objects that are not instances of the class.
 *
 * This is particularly useful when working with instances of classes that have
 * been duplicated or mangled by a bundler.
 *
 * @typeParam T - The type that extends Discriminated
 * @param discriminator - The string value to match against the object's discriminator property
 * @param cls - The class constructor to check for instanceof
 * @returns A type guard function that returns true if the value matches either condition
 *
 * @example
 * ```typescript
 * class User implements Discriminated {
 *   discriminator = "user";
 *   constructor(public name: string) {}
 * }
 *
 * const isUser = createMatcher("user", User);
 *
 * // Works with class instances
 * const user = new User("John");
 * if (isUser(user)) {
 *   console.log(user.name); // TypeScript knows user is a User
 * }
 *
 * // Works with plain objects
 * const obj = { discriminator: "user", name: "John" };
 * if (isUser(obj)) {
 *   console.log(obj.name); // TypeScript knows obj is a User
 * }
 * ```
 *
 * @example
 * ```typescript
 * // Working with multiple types
 * class Admin implements Discriminated {
 *   discriminator = "admin";
 *   constructor(public name: string, public role: string) {}
 * }
 *
 * class Guest implements Discriminated {
 *   discriminator = "guest";
 *   constructor(public name: string) {}
 * }
 *
 * const isAdmin = createMatcher("admin", Admin);
 * const isGuest = createMatcher("guest", Guest);
 *
 * function handleUser(user: unknown) {
 *   if (isAdmin(user)) {
 *     console.log(user.role); // TypeScript knows user is an Admin
 *   } else if (isGuest(user)) {
 *     console.log(user.name); // TypeScript knows user is a Guest
 *   }
 * }
 * ```
 */
export const createMatcher =
  <T extends Discriminated>(
    discriminator: string,
    cls: new (...args: any[]) => T,
  ): ((value: unknown) => value is T) =>
  (value: unknown): value is T =>
    value instanceof cls ||
    (typeof value === "object" &&
      value !== null &&
      "discriminator" in value &&
      value.discriminator === discriminator);
