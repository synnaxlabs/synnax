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

export const migratable = z.object({
  version: semVerZ,
});

export interface Migratable extends z.infer<typeof migratable> {}

export type Migration<I = unknown, O = unknown> = (
  migratable: Migratable & I,
) => Migratable & O;

export type Migrations = Record<string, Migration<any, any>>;

export const migrator = <I = unknown, O = unknown>(
  migrations: Migrations,
): Migration<I, O> => {
  const latestVersion = Object.keys(migrations).sort(compareSemVer).pop();
  if (latestVersion == null)
    return ((v: Migratable) => v) as unknown as Migration<I, O>;
  const migLength = Object.keys(migrations).length;
  const f = (old: Migratable): Migratable => {
    if (migLength === 0 || semVerNewer(old.version, latestVersion)) return old;
    const version = old.version;
    const migration = migrations[version];
    const new_: Migratable = migration(old);
    return f(new_);
  };
  return f as unknown as Migration<I, O>;
};
