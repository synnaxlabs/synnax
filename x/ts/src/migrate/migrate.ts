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

const compareSemVer: compare.CompareF<string> = (a, b) => {
  const semA = semVerZ.parse(a);
  const semB = semVerZ.parse(b);
  const [aMajor, aMinor, aPatch] = semA.split(".").map(Number);
  const [bMajor, bMinor, bPatch] = semB.split(".").map(Number);
  if (aMajor !== bMajor) return aMajor - bMajor;
  if (aMinor !== bMinor) return aMinor - bMinor;
  return aPatch - bPatch;
};

const semVerNewer = (a: SemVer, b: SemVer): boolean =>
  compare.isGreaterThan(compareSemVer(a, b));

export const migratable = (version?: SemVer) =>
  z.object({
    version: version ? z.literal(version) : semVerZ,
  });

export type Migratable<V extends string = string> = { version: V };

export type MigrationFunc<I extends z.ZodTypeAny, O extends z.ZodTypeAny> = (
  input: z.infer<I>,
) => z.infer<O>;

export interface Migration<I extends z.ZodTypeAny, O extends z.ZodTypeAny> {
  input: I;
  output: O;
  migrate: MigrationFunc<I, O>;
}

/**
 * A record of migrations to be applied, where the key of the record is the
 * input version of the migration.
 */
export type Migrations = Record<string, Migration<any, any>>;

interface MigratorProps<O extends z.ZodTypeAny> {
  name: string;
  migrations: Migrations;
  target: O;
  def: z.output<O>;
}

export const migrator = <O extends z.ZodTypeAny>({
  name,
  migrations,
  target,
  def,
}: MigratorProps<O>): ((v: Migratable) => z.output<O>) => {
  const latestVersion = Object.keys(migrations).sort(compareSemVer).pop();
  if (latestVersion == null)
    return ((v: Migratable) => target.parse(v)) as unknown as (
      v: Migratable,
    ) => z.output<O>;
  const migLength = Object.keys(migrations).length;
  const f = (old: Migratable): Migratable => {
    try {
      if (migLength === 0 || semVerNewer(old.version, latestVersion)) {
        console.log(
          `${name} version ${old.version} is newer than latest migration of ${latestVersion}`,
        );
        return old;
      }
      console.log(`${name} migrating from ${old.version}`);
      const version = old.version;
      const migration = migrations[version];
      // migration.input.parse(old);
      // if (!oldRes.success) {
      //   log?.(
      //     `${name} failed to parse old version ${old.version}. Trying to migrate anyway`,
      //   );
      //   log?.(oldRes.error.format());
      // }
      const new_: Migratable = migration.migrate(old);
      console.log(`${name} migrated to ${new_.version}`);
      // migration.output.parse(new_);
      // if (!newRes.success) {
      //   log?.(
      //     `${name} failed to parse new version ${latestVersion}. Exiting with default`,
      //   );
      //   log?.(newRes.error.format());
      //   return def;
      // }
      return f(new_);
    } catch (e) {
      console.log(`${name} failed to migrate from ${old.version} to ${latestVersion}`);
      console.error(e);
      return def;
    }
  };
  return (v: Migratable): z.output<O> => {
    try {
      return f(v) as z.output<O>;
      // return target.parse(f(v, opts));
      // if (!res.success) {
      //   log?.(`${name} failed to parse final result. Exiting with default`);
      //   log?.(res.error.format());
      //   return def;
      // }
      // return res.data;
    } catch (e) {
      console.log(`${name} failed to parse final result. Exiting with default`);
      console.error(e);
      return def;
    }
  };
};
