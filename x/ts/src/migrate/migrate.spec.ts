// // Copyright 2024 Synnax Labs, Inc.
// //
// // Use of this software is governed by the Business Source License included in the file
// // licenses/BSL.txt.
// //
// // As of the Change Date specified in that file, in accordance with the Business Source
// // License, use of this software will be governed by the Apache License, Version 2.0,
// // included in the file licenses/APL.txt.

// import { describe, expect, it } from "vitest";
// import { z } from "zod";

// import {
//   migratable,
//   type Migration,
//   type Migrations,
//   migrator,
// } from "@/migrate/migrate";

// const entityV0_0_0 = migratable.extend({
//   name: z.string(),
// });

// type EntityV0_0_0 = z.infer<typeof entityV0_0_0>;

// const entityV0_0_1 = migratable.extend({
//   title: z.string(),
// });

// type EntityV0_0_1 = z.infer<typeof entityV0_0_1>;

// const entityV0_0_2 = migratable.extend({
//   title: z.string(),
//   description: z.string(),
// });

// type EntityV0_0_2 = z.infer<typeof entityV0_0_2>;

// const migrations: Migrations = {
//   "0.0.0": ((entity: EntityV0_0_0): EntityV0_0_1 => {
//     const { name, ...rest } = entity;
//     return {
//       ...rest,
//       version: "0.0.1",
//       title: entity.name,
//     };
//   }) as Migration<EntityV0_0_0, EntityV0_0_1>,
//   "0.0.1": (entity: EntityV0_0_1): EntityV0_0_2 => {
//     return {
//       ...entity,
//       version: "0.0.2",
//       description: "",
//     };
//   },
// };

// describe("migrator", () => {
//   it("should migrate an entity from v0.0.0 to v0.0.2", () => {
//     const entity: EntityV0_0_0 = {
//       version: "0.0.0",
//       name: "foo",
//     };
//     const migrated = migrator({ name: "example", migrations })(entity);
//     expect(migrated).toEqual({
//       version: "0.0.2",
//       title: "foo",
//       description: "",
//     });
//   });
//   it("should not migrate an entity from v0.0.2", () => {
//     const entity: EntityV0_0_2 = {
//       version: "0.0.2",
//       title: "foo",
//       description: "bar",
//     };
//     const migrated = migrator({ name: "example", migrations })(entity);
//     expect(migrated).toEqual(entity);
//   });
//   it("should not migrate an entity from v0.0.3", () => {
//     const entity = {
//       version: "0.0.3",
//       title: "foo",
//       description: "bar",
//     };
//     const migrated = migrator({ name: "example", migrations })(entity);
//     expect(migrated).toEqual(entity);
//   });
// });
