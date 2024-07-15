// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { migrate } from "@/migrate";

const entityV0 = z.object({
  version: z.literal("0.0.0"),
  name: z.string(),
});

type EntityV0 = z.infer<typeof entityV0>;

const entityV1 = z.object({
  version: z.literal("1.0.0"),
  title: z.string(),
});

const migrateV1 = migrate.createMigration<EntityV0, EntityV1>({
  name: "entity",
  inputSchema: entityV0,
  outputSchema: entityV1,
  migrate: (entity) => {
    const { name, ...rest } = entity;
    return { ...rest, version: "1.0.0", title: entity.name };
  },
});

type EntityV1 = z.infer<typeof entityV1>;

const entityV2 = z.object({
  version: z.literal("2.0.0"),
  title: z.string(),
  description: z.string(),
});

type EntityV2 = z.infer<typeof entityV2>;

const migrateV2 = migrate.createMigration<EntityV1, EntityV2>({
  name: "entity",
  inputSchema: entityV1,
  outputSchema: entityV2,
  migrate: (entity) => ({ ...entity, version: "2.0.0", description: "" }),
});

const migrations: migrate.Migrations = {
  "0.0.0": migrateV1,
  "1.0.0": migrateV2,
};

describe("compareSemVer", () => {
  it("should return true when the major version is higher", () => {
    expect(migrate.compareSemVer("1.0.0", "0.0.0")).toBeGreaterThan(0);
    expect(migrate.semVerNewer("3.0.0", "0.3.0")).toBeTruthy();
  });
});

describe("migrator", () => {
  it("should migrate an entity from v0 to v2", () => {
    const entity: EntityV0 = { version: "0.0.0", name: "foo" };
    const DEFAULT: EntityV2 = { version: "2.0.0", title: "", description: "" };
    const migrated = migrate.migrator({
      name: "entity",
      migrations,
      def: DEFAULT,
    })(entity);
    expect(migrated).toEqual({ version: "2.0.0", title: "foo", description: "" });
  });
  it("should migrate an entity from v1 to v2", () => {
    const entity: EntityV1 = { version: "1.0.0", title: "foo" };
    const DEFAULT: EntityV2 = { version: "2.0.0", title: "", description: "" };
    const migrated = migrate.migrator({
      name: "entity",
      migrations,
      def: DEFAULT,
    })(entity);
    expect(migrated).toEqual({ version: "2.0.0", title: "foo", description: "" });
  });
  it("should not migrate an entity from v2 to v2", () => {
    const entity: EntityV2 = { version: "2.0.0", title: "foo", description: "bar" };
    const DEFAULT: EntityV2 = { version: "2.0.0", title: "", description: "" };
    const migrated = migrate.migrator({
      name: "entity",
      migrations,
      def: DEFAULT,
    })(entity);
    expect(migrated).toEqual(entity);
  });
});
