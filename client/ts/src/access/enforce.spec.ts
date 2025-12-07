// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { access } from "@/access";
import { type Policy } from "@/access/policy/payload";
import { type ontology } from "@/ontology";

const id = (type: ontology.ResourceType, key: string): ontology.ID => ({ type, key });

const policy = (
  objects: ontology.ID[],
  actions: access.Action[],
  key = crypto.randomUUID(),
): Policy => ({ key, name: "test", objects, actions, internal: false });

describe("allowRequest", () => {
  describe("single object", () => {
    it("should allow when policy has exact match", () => {
      const policies = [policy([id("channel", "1")], ["retrieve"])];
      const allowed = access.allowRequest(
        {
          subject: id("user", "u1"),
          action: "retrieve",
          objects: id("channel", "1"),
        },
        policies,
      );
      expect(allowed).toBe(true);
    });

    it("should allow when policy has type-level match (empty key)", () => {
      const policies = [policy([id("channel", "")], ["retrieve"])];
      const allowed = access.allowRequest(
        {
          subject: id("user", "u1"),
          action: "retrieve",
          objects: id("channel", "42"),
        },
        policies,
      );
      expect(allowed).toBe(true);
    });

    it("should deny when no policy matches", () => {
      const policies = [policy([id("channel", "1")], ["retrieve"])];
      const allowed = access.allowRequest(
        {
          subject: id("user", "u1"),
          action: "retrieve",
          objects: id("channel", "2"),
        },
        policies,
      );
      expect(allowed).toBe(false);
    });

    it("should deny when action not allowed", () => {
      const policies = [policy([id("channel", "1")], ["retrieve"])];
      const allowed = access.allowRequest(
        {
          subject: id("user", "u1"),
          action: "delete",
          objects: id("channel", "1"),
        },
        policies,
      );
      expect(allowed).toBe(false);
    });

    it("should deny when type does not match", () => {
      const policies = [policy([id("channel", "1")], ["retrieve"])];
      const allowed = access.allowRequest(
        {
          subject: id("user", "u1"),
          action: "retrieve",
          objects: id("device", "1"),
        },
        policies,
      );
      expect(allowed).toBe(false);
    });
  });

  describe("multiple objects", () => {
    it("should allow when all objects are covered", () => {
      const policies = [policy([id("channel", "1"), id("channel", "2")], ["retrieve"])];
      const allowed = access.allowRequest(
        {
          subject: id("user", "u1"),
          action: "retrieve",
          objects: [id("channel", "1"), id("channel", "2")],
        },
        policies,
      );
      expect(allowed).toBe(true);
    });

    it("should deny when some objects are not covered", () => {
      const policies = [policy([id("channel", "1")], ["retrieve"])];
      const allowed = access.allowRequest(
        {
          subject: id("user", "u1"),
          action: "retrieve",
          objects: [id("channel", "1"), id("channel", "2")],
        },
        policies,
      );
      expect(allowed).toBe(false);
    });

    it("should allow all objects with type-level policy", () => {
      const policies = [policy([id("channel", "")], ["retrieve"])];
      const allowed = access.allowRequest(
        {
          subject: id("user", "u1"),
          action: "retrieve",
          objects: [id("channel", "1"), id("channel", "2"), id("channel", "99")],
        },
        policies,
      );
      expect(allowed).toBe(true);
    });
  });

  describe("multiple policies", () => {
    it("should allow when different policies cover different objects", () => {
      const policies = [
        policy([id("channel", "1")], ["retrieve"]),
        policy([id("channel", "2")], ["retrieve"]),
      ];
      const allowed = access.allowRequest(
        {
          subject: id("user", "u1"),
          action: "retrieve",
          objects: [id("channel", "1"), id("channel", "2")],
        },
        policies,
      );
      expect(allowed).toBe(true);
    });

    it("should allow when one policy covers object and another has different action", () => {
      const policies = [
        policy([id("channel", "1")], ["delete"]),
        policy([id("channel", "1")], ["retrieve"]),
      ];
      const allowed = access.allowRequest(
        {
          subject: id("user", "u1"),
          action: "retrieve",
          objects: id("channel", "1"),
        },
        policies,
      );
      expect(allowed).toBe(true);
    });
  });

  describe("edge cases", () => {
    it("should deny with empty policies", () => {
      const allowed = access.allowRequest(
        {
          subject: id("user", "u1"),
          action: "retrieve",
          objects: id("channel", "1"),
        },
        [],
      );
      expect(allowed).toBe(false);
    });

    it("should allow with empty objects", () => {
      const policies = [policy([id("channel", "1")], ["retrieve"])];
      const allowed = access.allowRequest(
        { subject: id("user", "u1"), action: "retrieve", objects: [] },
        policies,
      );
      expect(allowed).toBe(true);
    });
  });
});
