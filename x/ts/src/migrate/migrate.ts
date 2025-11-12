// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { compare } from "@/compare";
import { type Optional } from "@/optional";

export const semVerZ = z
  .string()
  .regex(/^\d+\.\d+\.\d+(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$/);

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
 * Compares two pre-release identifiers according to semver spec.
 * @param a - First pre-release identifier (without leading hyphen)
 * @param b - Second pre-release identifier (without leading hyphen)
 * @returns compare.LESS_THAN if a < b, compare.GREATER_THAN if a > b, compare.EQUAL if equal
 */
const comparePreRelease = (a: string, b: string): number => {
  const aParts = a.split(".");
  const bParts = b.split(".");
  const maxLength = Math.max(aParts.length, bParts.length);

  for (let i = 0; i < maxLength; i++) {
    const aPart = aParts[i];
    const bPart = bParts[i];

    // A larger set of pre-release fields has higher precedence
    if (aPart === undefined) return compare.LESS_THAN;
    if (bPart === undefined) return compare.GREATER_THAN;

    const aIsNumeric = /^\d+$/.test(aPart);
    const bIsNumeric = /^\d+$/.test(bPart);

    // Numeric identifiers always have lower precedence than non-numeric
    if (aIsNumeric && !bIsNumeric) return compare.LESS_THAN;
    if (!aIsNumeric && bIsNumeric) return compare.GREATER_THAN;

    if (aIsNumeric && bIsNumeric) {
      // Compare numerically
      const aNum = parseInt(aPart, 10);
      const bNum = parseInt(bPart, 10);
      if (aNum < bNum) return compare.LESS_THAN;
      if (aNum > bNum) return compare.GREATER_THAN;
    } else {
      // Compare lexically (ASCII sort order)
      if (aPart < bPart) return compare.LESS_THAN;
      if (aPart > bPart) return compare.GREATER_THAN;
    }
  }

  return compare.EQUAL;
};

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

  // Split version and pre-release parts
  const [aCore, aPreRelease] = semA.split("-");
  const [bCore, bPreRelease] = semB.split("-");

  const [aMajor, aMinor, aPatch] = aCore.split(".").map(Number);
  const [bMajor, bMinor, bPatch] = bCore.split(".").map(Number);

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

  // When major.minor.patch are equal, compare pre-release versions
  // Version without pre-release > version with pre-release
  if (aPreRelease === undefined && bPreRelease === undefined) return compare.EQUAL;
  if (aPreRelease === undefined) return compare.GREATER_THAN;
  if (bPreRelease === undefined) return compare.LESS_THAN;

  // Both have pre-release, compare them
  return comparePreRelease(aPreRelease, bPreRelease);
}) satisfies compare.Comparator<SemVer>;

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
  ZI extends z.ZodType,
  ZO extends z.ZodType,
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
    ZI extends z.ZodType = z.ZodType,
    ZO extends z.ZodType = z.ZodType,
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

interface MigratorProps<O extends Migratable, ZO extends z.ZodType = z.ZodType> {
  name: string;
  migrations: Migrations;
  def: O;
  defaultVersion?: string;
  targetSchema?: ZO;
}

export type Migrator = <I extends Optional<Migratable, "version">, O>(v: I) => O;

export const migrator = <
  I extends Optional<Migratable, "version">,
  O extends Migratable,
  ZO extends z.ZodType = z.ZodType,
>({
  name,
  migrations,
  targetSchema,
  defaultVersion,
  def,
}: MigratorProps<O, ZO>): ((v: I) => O) => {
  const latestMigrationVersion = Object.keys(migrations).sort(compareSemVer).pop();
  if (latestMigrationVersion == null)
    return (v: I) => {
      v.version ??= defaultVersion;
      if (v.version !== def.version) {
        console.log(
          `${name} version ${v.version} is newer than latest version of ${def.version}.
          Returning default instead.
          `,
        );
        return def;
      }
      try {
        if (targetSchema != null) return targetSchema.parse(v) as O;
        return v as unknown as O;
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
  return (v: I): O => {
    try {
      if (v.version == null)
        if (defaultVersion != null) {
          console.log(
            `${name} version is null. Setting version to default of ${defaultVersion}`,
          );
          v.version = defaultVersion;
        } else {
          console.log(
            `${name} version is null and no default version set. Exiting with default`,
          );
          return def;
        }
      return f(v as Migratable) as O;
    } catch (e) {
      console.log(`${name} failed to parse final result. Exiting with default`);
      console.error(e);
      return def;
    }
  };
};
