// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, it, expect } from "vitest";

import { ontology } from "@/ontology";
import { newClient } from "@/setupspecs";

const client = newClient();

describe("Group", () => {
  describe("create", () => {
    it("should correctly create a group", async () => {
      const name = `group-${Math.random()}`;
      const g = await client.ontology.groups.create(ontology.Root, name);
      expect(g.name).toEqual(name);
    });
    it("should correctly rename a group", async () => {
      const name = `group-${Math.random()}`;
      const g = await client.ontology.groups.create(ontology.Root, name);
      const newName = `group-${Math.random()}`;
      await client.ontology.groups.rename(g.key, newName);
      const g2 = await client.ontology.retrieve(g.ontologyID);
      expect(g2.name).toEqual(newName);
    });
  });
});
