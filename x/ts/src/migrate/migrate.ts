// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { compare } from "@/compare";

export const semVerZ = z.string().regex(/^\d+\.\d+\.\d+$/);

export type SemVer = z.infer<typeof semVerZ>;

export interface CompareSemVerOptions {
  /**
   * Whether to validate that major versions are equal.
   * @default true
   */
  checkMajor?: boolean;
  /**
   * Whether to validate that minor versions are equal.
   * @default true
   */
  checkMinor?: boolean;
  /**
   * Whether to validate that patch versions are equal.
   * @default true
   */
  checkPatch?: boolean;
}

/**
 * Compares the two semantic versions.
 *
 * @param a  The first semantic version.
 * @param b  The second semantic version.
 * @param opts - Optional object to disable checking specific version parts
 * (major, minor, patch).
 * @returns a number, where the the number is compare.LESS_THAN (negative) if a is OLDER
 * than B, compare.EQUAL (0) if a is the same as b, and compare.GREATER_THAN (positive)
 * if a is NEWER than b.
 */
export const compareSemVer = ((
  a: SemVer,
  b: SemVer,
  opts: CompareSemVerOptions = {},
) => {
  opts.checkMajor ??= true;
  opts.checkMinor ??= true;
  opts.checkPatch ??= true;
  const semA = semVerZ.parse(a);
  const semB = semVerZ.parse(b);
  const [aMajor, aMinor, aPatch] = semA.split(".").map(Number);
  const [bMajor, bMinor, bPatch] = semB.split(".").map(Number);
  if (opts.checkMajor) {
    if (aMajor < bMajor) return compare.LESS_THAN;
    if (aMajor > bMajor) return compare.GREATER_THAN;
  }
  if (opts.checkMinor) {
    if (aMinor < bMinor) return compare.LESS_THAN;
    if (aMinor > bMinor) return compare.GREATER_THAN;
  }
  if (opts.checkPatch) {
    if (aPatch < bPatch) return compare.LESS_THAN;
    if (aPatch > bPatch) return compare.GREATER_THAN;
  }
  return compare.EQUAL;
}) satisfies compare.CompareF<SemVer>;

/**
 * @returns true if the two semantic versions are equal.
 * @param a - The first semantic version.
 * @param b - The second semantic version.
 * @param opts - Optional object to disable checking specific version parts
 * (major, minor, patch).
 */
export const versionsEqual = (
  a: SemVer,
  b: SemVer,
  opts: CompareSemVerOptions = {},
): boolean => compare.isEqualTo(compareSemVer(a, b, opts));

/**
 * @returns true if the first semantic version is newer than the second.
 * @param a The first semantic version.
 * @param b The second semantic version.
 * @param opts - Optional object to disable checking specific version parts
 * (major, minor, patch).
 */
export const semVerNewer = (
  a: SemVer,
  b: SemVer,
  opts: CompareSemVerOptions = {},
): boolean => compare.isGreaterThan(compareSemVer(a, b, opts));

/**
 * @returns true if the first semantic version is older than the second.
 * @param a The first semantic version.
 * @param b The second semantic version.
 * @param opts - Optional object to disable checking specific version parts
 * (major, minor, patch).
 */
export const semVerOlder = (
  a: SemVer,
  b: SemVer,
  opts: CompareSemVerOptions = {},
): boolean => compare.isLessThan(compareSemVer(a, b, opts));

export type Migratable<V extends string = string> = { version: V };

export type Migration<I extends Migratable, O extends Migratable> = (input: I) => O;

export interface MigrationProps<
  I extends Migratable,
  O extends Migratable,
  ZI extends z.ZodTypeAny,
  ZO extends z.ZodTypeAny,
> {
  name: string;
  inputSchema?: ZI;
  outputSchema?: ZO;
  migrate: Migration<I, O>;
}

export const createMigration =
  <
    I extends Migratable,
    O extends Migratable,
    ZI extends z.ZodTypeAny = z.ZodTypeAny,
    ZO extends z.ZodTypeAny = z.ZodTypeAny,
  >({
    name,
    migrate,
  }: MigrationProps<I, O, ZI, ZO>): Migration<I, O> =>
  (input: I): O => {
    try {
      const out = migrate(input);
      console.log(`${name} migrated: ${input.version} -> ${out.version}`);
      return out;
    } catch (e) {
      console.log(`${name} failed to migrate from ${input.version}`);
      console.error(e);
      throw e;
    }
  };

/**
 * A record of migrations to be applied, where the key of the record is the
 * input version of the migration.
 */
export type Migrations = Record<string, Migration<any, any>>;

interface MigratorProps<O extends Migratable, ZO extends z.ZodTypeAny = z.ZodTypeAny> {
  name: string;
  migrations: Migrations;
  def: O;
  targetSchema?: ZO;
}

export const migrator = <
  I extends Migratable,
  O extends Migratable,
  ZO extends z.ZodTypeAny = z.ZodTypeAny,
>({
  name,
  migrations,
  targetSchema,
  def,
}: MigratorProps<O, ZO>): ((v: I) => O) => {
  const latestMigrationVersion = Object.keys(migrations).sort(compareSemVer).pop();
  if (latestMigrationVersion == null)
    return (v: Migratable) => {
      if (v.version !== def.version) {
        console.log(
          `${name} version ${v.version} is newer than latest version of ${def.version}. 
          Returning default instead.
          `,
        );
        return def;
      }
      try {
        if (targetSchema != null) return targetSchema.parse(v);
        return v;
      } catch (e) {
        console.log(`${name} failed to parse default. Exiting with default`);
        console.error(e);
        return def;
      }
    };
  const migLength = Object.keys(migrations).length;
  let migrationApplied = false;
  const f = (old: Migratable): Migratable => {
    try {
      if (migLength === 0 || semVerNewer(old.version, latestMigrationVersion)) {
        if (migrationApplied) console.log(`${name} ${old.version} now up to date`);
        else
          console.log(
            `${name} version ${old.version} is up to date with target version ${def.version}`,
          );
        return old;
      }
      const version = old.version;
      const migrate = migrations[version];
      const new_: Migratable = migrate(old);
      migrationApplied = true;
      return f(new_);
    } catch (e) {
      console.log(
        `${name} failed to migrate from ${old.version} to ${latestMigrationVersion}`,
      );
      console.error(e);
      return def;
    }
  };
  return (v: Migratable): O => {
    try {
      return f(v) as O;
    } catch (e) {
      console.log(`${name} failed to parse final result. Exiting with default`);
      console.error(e);
      return def;
    }
  };
};
