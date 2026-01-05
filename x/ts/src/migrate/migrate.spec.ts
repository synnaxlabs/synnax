// Copyright 2026 Synnax Labs, Inc.
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

describe("semVerZ", () => {
  describe("valid versions", () => {
    it("should accept standard semver format", () => {
      expect(() => migrate.semVerZ.parse("1.0.0")).not.toThrow();
      expect(() => migrate.semVerZ.parse("0.0.0")).not.toThrow();
      expect(() => migrate.semVerZ.parse("99.99.99")).not.toThrow();
    });

    it("should accept pre-release versions with single identifier", () => {
      expect(() => migrate.semVerZ.parse("1.0.0-alpha")).not.toThrow();
      expect(() => migrate.semVerZ.parse("1.0.0-beta")).not.toThrow();
      expect(() => migrate.semVerZ.parse("1.0.0-rc")).not.toThrow();
      expect(() => migrate.semVerZ.parse("0.48.0-rc")).not.toThrow();
    });

    it("should accept pre-release versions with numeric identifiers", () => {
      expect(() => migrate.semVerZ.parse("1.0.0-1")).not.toThrow();
      expect(() => migrate.semVerZ.parse("1.0.0-0")).not.toThrow();
      expect(() => migrate.semVerZ.parse("1.0.0-99")).not.toThrow();
    });

    it("should accept pre-release versions with multiple identifiers", () => {
      expect(() => migrate.semVerZ.parse("1.0.0-alpha.1")).not.toThrow();
      expect(() => migrate.semVerZ.parse("1.0.0-rc.1")).not.toThrow();
      expect(() => migrate.semVerZ.parse("1.0.0-beta.2.3")).not.toThrow();
      expect(() => migrate.semVerZ.parse("1.0.0-0.3.7")).not.toThrow();
    });

    it("should accept pre-release versions with hyphens", () => {
      expect(() => migrate.semVerZ.parse("1.0.0-x-beta")).not.toThrow();
      expect(() => migrate.semVerZ.parse("1.0.0-alpha-beta")).not.toThrow();
    });
  });

  describe("invalid versions", () => {
    it("should reject versions without patch", () => {
      expect(() => migrate.semVerZ.parse("1.0")).toThrow();
    });

    it("should reject versions without minor", () => {
      expect(() => migrate.semVerZ.parse("1")).toThrow();
    });

    it("should reject versions with build metadata (not supported)", () => {
      expect(() => migrate.semVerZ.parse("1.0.0+build")).toThrow();
    });

    it("should reject versions with empty pre-release", () => {
      expect(() => migrate.semVerZ.parse("1.0.0-")).toThrow();
    });
  });
});

describe("compareSemVer", () => {
  it("should return true when the major version is higher", () => {
    expect(migrate.compareSemVer("1.0.0", "0.0.0")).toBeGreaterThan(0);
    expect(migrate.semVerNewer("3.0.0", "0.3.0")).toBe(true);
  });
  describe("only check major", () => {
    it("should return equal when the major versions are equal but the minor and patch are different", () => {
      expect(
        migrate.compareSemVer("1.0.0", "1.1.0", {
          checkMinor: false,
          checkPatch: false,
        }),
      ).toBe(0);
    });
    it("should return greater than when the major version is higher", () => {
      expect(
        migrate.compareSemVer("2.0.0", "1.1.0", {
          checkMinor: false,
          checkPatch: false,
        }),
      ).toBeGreaterThan(0);
    });
    it("should return less than when the major version is lower", () => {
      expect(
        migrate.compareSemVer("1.0.0", "2.1.0", {
          checkMinor: false,
          checkPatch: false,
        }),
      ).toBeLessThan(0);
    });
  });
  describe("only check minor", () => {
    it("should return equal when the minor versions are equal but the major and patch are different", () => {
      expect(
        migrate.compareSemVer("1.0.0", "2.0.0", {
          checkMajor: false,
          checkPatch: false,
        }),
      ).toBe(0);
    });
    it("should return greater than when the minor version is higher", () => {
      expect(
        migrate.compareSemVer("1.2.0", "1.1.0", {
          checkMajor: false,
          checkPatch: false,
        }),
      ).toBeGreaterThan(0);
    });
    it("should return less than when the minor version is lower", () => {
      expect(
        migrate.compareSemVer("1.0.0", "1.1.0", {
          checkMajor: false,
          checkPatch: false,
        }),
      ).toBeLessThan(0);
    });
  });
  describe("only check patch", () => {
    it("should return equal when the patch versions are equal but the major and minor are different", () => {
      expect(
        migrate.compareSemVer("2.1.1", "2.2.1", {
          checkMajor: false,
          checkMinor: false,
        }),
      ).toBe(0);
    });
    it("should return greater than when the patch version is higher", () => {
      expect(
        migrate.compareSemVer("1.4.2", "1.9.1", {
          checkMajor: false,
          checkMinor: false,
        }),
      ).toBeGreaterThan(0);
    });
    it("should return less than when the patch version is lower", () => {
      expect(
        migrate.compareSemVer("10000.2.0", "95.6.1", {
          checkMajor: false,
          checkMinor: false,
        }),
      ).toBeLessThan(0);
    });
  });

  describe("pre-release versions", () => {
    it("should consider release version newer than pre-release", () => {
      expect(migrate.compareSemVer("1.0.0", "1.0.0-rc")).toBeGreaterThan(0);
      expect(migrate.compareSemVer("1.0.0", "1.0.0-alpha")).toBeGreaterThan(0);
      expect(migrate.compareSemVer("1.0.0", "1.0.0-beta")).toBeGreaterThan(0);
      expect(migrate.compareSemVer("0.48.0", "0.48.0-rc")).toBeGreaterThan(0);
      expect(migrate.semVerNewer("1.0.0", "1.0.0-rc")).toBe(true);
    });

    it("should consider pre-release version older than release", () => {
      expect(migrate.compareSemVer("1.0.0-rc", "1.0.0")).toBeLessThan(0);
      expect(migrate.compareSemVer("1.0.0-alpha", "1.0.0")).toBeLessThan(0);
      expect(migrate.compareSemVer("0.48.0-rc", "0.48.0")).toBeLessThan(0);
      expect(migrate.semVerOlder("1.0.0-rc", "1.0.0")).toBe(true);
    });

    it("should compare pre-release versions lexicographically", () => {
      expect(migrate.compareSemVer("1.0.0-alpha", "1.0.0-beta")).toBeLessThan(0);
      expect(migrate.compareSemVer("1.0.0-beta", "1.0.0-rc")).toBeLessThan(0);
      expect(migrate.compareSemVer("1.0.0-rc", "1.0.0-alpha")).toBeGreaterThan(0);
      expect(migrate.semVerNewer("1.0.0-rc", "1.0.0-alpha")).toBe(true);
      expect(migrate.semVerOlder("1.0.0-alpha", "1.0.0-beta")).toBe(true);
    });

    it("should compare numeric pre-release identifiers numerically", () => {
      expect(migrate.compareSemVer("1.0.0-1", "1.0.0-2")).toBeLessThan(0);
      expect(migrate.compareSemVer("1.0.0-2", "1.0.0-10")).toBeLessThan(0);
      expect(migrate.compareSemVer("1.0.0-10", "1.0.0-2")).toBeGreaterThan(0);
    });

    it("should compare pre-release versions with multiple identifiers", () => {
      expect(migrate.compareSemVer("1.0.0-rc.1", "1.0.0-rc.2")).toBeLessThan(0);
      expect(migrate.compareSemVer("1.0.0-rc.2", "1.0.0-rc.10")).toBeLessThan(0);
      expect(migrate.compareSemVer("1.0.0-alpha.1", "1.0.0-alpha.2")).toBeLessThan(0);
      expect(
        migrate.compareSemVer("1.0.0-alpha.beta", "1.0.0-alpha.gamma"),
      ).toBeLessThan(0);
    });

    it("should consider numeric identifiers lower than alphanumeric", () => {
      expect(migrate.compareSemVer("1.0.0-1", "1.0.0-alpha")).toBeLessThan(0);
      expect(migrate.compareSemVer("1.0.0-alpha", "1.0.0-1")).toBeGreaterThan(0);
      expect(migrate.compareSemVer("1.0.0-rc.1", "1.0.0-rc.beta")).toBeLessThan(0);
    });

    it("should consider longer pre-release identifiers higher precedence", () => {
      expect(migrate.compareSemVer("1.0.0-rc", "1.0.0-rc.1")).toBeLessThan(0);
      expect(migrate.compareSemVer("1.0.0-rc.1", "1.0.0-rc")).toBeGreaterThan(0);
      expect(migrate.compareSemVer("1.0.0-alpha.1", "1.0.0-alpha.1.2")).toBeLessThan(0);
    });

    it("should consider equal pre-release versions equal", () => {
      expect(migrate.compareSemVer("1.0.0-rc", "1.0.0-rc")).toBe(0);
      expect(migrate.compareSemVer("1.0.0-alpha.1", "1.0.0-alpha.1")).toBe(0);
      expect(migrate.versionsEqual("1.0.0-rc", "1.0.0-rc")).toBe(true);
    });

    it("should handle complex pre-release comparison chains", () => {
      const versions = [
        "1.0.0-alpha",
        "1.0.0-alpha.1",
        "1.0.0-alpha.beta",
        "1.0.0-beta",
        "1.0.0-beta.2",
        "1.0.0-beta.11",
        "1.0.0-rc.1",
        "1.0.0",
      ];

      for (let i = 0; i < versions.length - 1; i++) {
        expect(migrate.compareSemVer(versions[i], versions[i + 1])).toBeLessThan(0);
        expect(migrate.compareSemVer(versions[i + 1], versions[i])).toBeGreaterThan(0);
      }
    });

    it("should respect checkMajor/checkMinor/checkPatch with pre-release", () => {
      expect(
        migrate.compareSemVer("2.0.0-rc", "1.0.0", {
          checkMinor: false,
          checkPatch: false,
        }),
      ).toBeGreaterThan(0);

      expect(
        migrate.compareSemVer("1.2.0-rc", "1.1.0", {
          checkMajor: false,
          checkPatch: false,
        }),
      ).toBeGreaterThan(0);
    });
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

  describe("with pre-release versions", () => {
    const entityV1RC = z.object({
      version: z.literal("1.0.0-rc"),
      title: z.string(),
    });

    type EntityV1RC = z.infer<typeof entityV1RC>;

    const migrateV1RC = migrate.createMigration<EntityV1RC, EntityV2>({
      name: "entity",
      inputSchema: entityV1RC,
      outputSchema: entityV2,
      migrate: (entity) => ({ ...entity, version: "2.0.0", description: "" }),
    });

    const migrationsWithRC: migrate.Migrations = {
      "0.0.0": migrateV1,
      "1.0.0-rc": migrateV1RC,
      "1.0.0": migrateV2,
    };

    it("should migrate from pre-release version to stable", () => {
      const entity: EntityV1RC = { version: "1.0.0-rc", title: "foo" };
      const DEFAULT: EntityV2 = { version: "2.0.0", title: "", description: "" };
      const migrated = migrate.migrator({
        name: "entity",
        migrations: migrationsWithRC,
        def: DEFAULT,
      })(entity);
      expect(migrated).toEqual({ version: "2.0.0", title: "foo", description: "" });
    });

    it("should handle version sorting with pre-release correctly", () => {
      const versions = ["0.0.0", "1.0.0-rc", "1.0.0"];
      const sorted = versions.sort(migrate.compareSemVer);
      expect(sorted).toEqual(["0.0.0", "1.0.0-rc", "1.0.0"]);
    });

    it("should not migrate if current version is newer than pre-release target", () => {
      const entity: EntityV1 = { version: "1.0.0", title: "foo" };
      const DEFAULT: EntityV1RC = { version: "1.0.0-rc", title: "" };
      const migrated = migrate.migrator({
        name: "entity",
        migrations: { "0.0.0": migrateV1 },
        def: DEFAULT,
      })(entity);
      expect(migrated).toEqual({ version: "1.0.0", title: "foo" });
    });

    it("should migrate through multiple pre-release versions", () => {
      interface EntityV1Alpha {
        version: "1.0.0-alpha";
        title: string;
      }

      interface EntityV1Beta extends Omit<EntityV1Alpha, "version"> {
        version: "1.0.0-beta";
        newField: string;
      }

      const migrateAlphaToBeta = migrate.createMigration<EntityV1Alpha, EntityV1Beta>({
        name: "entity",
        migrate: (entity) => ({
          ...entity,
          version: "1.0.0-beta",
          newField: "added",
        }),
      });

      const migrateBetaToRC = migrate.createMigration<EntityV1Beta, EntityV1RC>({
        name: "entity",
        migrate: (entity) => {
          const { newField, ...rest } = entity;
          return { ...rest, version: "1.0.0-rc" };
        },
      });

      const preReleaseMigrations: migrate.Migrations = {
        "1.0.0-alpha": migrateAlphaToBeta,
        "1.0.0-beta": migrateBetaToRC,
      };

      const entity: EntityV1Alpha = { version: "1.0.0-alpha", title: "test" };
      const DEFAULT: EntityV1RC = { version: "1.0.0-rc", title: "" };
      const migrated = migrate.migrator({
        name: "entity",
        migrations: preReleaseMigrations,
        def: DEFAULT,
      })(entity);
      expect(migrated).toEqual({ version: "1.0.0-rc", title: "test" });
    });
  });
});
