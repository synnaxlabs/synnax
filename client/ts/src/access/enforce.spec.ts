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
      const policies = [policy([id("channel", "1")], [access.RETRIEVE_ACTION])];
      const allowed = access.allowRequest(
        {
          subject: id("user", "u1"),
          actions: access.RETRIEVE_ACTION,
          objects: id("channel", "1"),
        },
        policies,
      );
      expect(allowed).toBe(true);
    });

    it("should allow when policy has type-level match (empty key)", () => {
      const policies = [policy([id("channel", "")], [access.RETRIEVE_ACTION])];
      const allowed = access.allowRequest(
        {
          subject: id("user", "u1"),
          actions: access.RETRIEVE_ACTION,
          objects: id("channel", "42"),
        },
        policies,
      );
      expect(allowed).toBe(true);
    });

    it("should deny when no policy matches", () => {
      const policies = [policy([id("channel", "1")], [access.RETRIEVE_ACTION])];
      const allowed = access.allowRequest(
        {
          subject: id("user", "u1"),
          actions: access.RETRIEVE_ACTION,
          objects: id("channel", "2"),
        },
        policies,
      );
      expect(allowed).toBe(false);
    });

    it("should deny when action not allowed", () => {
      const policies = [policy([id("channel", "1")], [access.RETRIEVE_ACTION])];
      const allowed = access.allowRequest(
        {
          subject: id("user", "u1"),
          actions: access.DELETE_ACTION,
          objects: id("channel", "1"),
        },
        policies,
      );
      expect(allowed).toBe(false);
    });

    it("should deny when type does not match", () => {
      const policies = [policy([id("channel", "1")], [access.RETRIEVE_ACTION])];
      const allowed = access.allowRequest(
        {
          subject: id("user", "u1"),
          actions: access.RETRIEVE_ACTION,
          objects: id("device", "1"),
        },
        policies,
      );
      expect(allowed).toBe(false);
    });
  });

  describe("wildcard action", () => {
    it("should allow any action when policy has ALL_ACTION", () => {
      const policies = [policy([id("channel", "1")], [access.ALL_ACTION])];
      expect(
        access.allowRequest(
          {
            subject: id("user", "u1"),
            actions: access.RETRIEVE_ACTION,
            objects: id("channel", "1"),
          },
          policies,
        ),
      ).toBe(true);
      expect(
        access.allowRequest(
          {
            subject: id("user", "u1"),
            actions: access.DELETE_ACTION,
            objects: id("channel", "1"),
          },
          policies,
        ),
      ).toBe(true);
      expect(
        access.allowRequest(
          {
            subject: id("user", "u1"),
            actions: access.CREATE_ACTION,
            objects: id("channel", "1"),
          },
          policies,
        ),
      ).toBe(true);
    });
  });

  describe("multiple actions", () => {
    it("should allow when policy covers all requested actions", () => {
      const policies = [
        policy([id("channel", "1")], [access.RETRIEVE_ACTION, access.DELETE_ACTION]),
      ];
      const allowed = access.allowRequest(
        {
          subject: id("user", "u1"),
          actions: [access.RETRIEVE_ACTION, access.DELETE_ACTION],
          objects: id("channel", "1"),
        },
        policies,
      );
      expect(allowed).toBe(true);
    });

    it("should deny when policy only covers some requested actions", () => {
      const policies = [policy([id("channel", "1")], [access.RETRIEVE_ACTION])];
      const allowed = access.allowRequest(
        {
          subject: id("user", "u1"),
          actions: [access.RETRIEVE_ACTION, access.DELETE_ACTION],
          objects: id("channel", "1"),
        },
        policies,
      );
      expect(allowed).toBe(false);
    });

    it("should allow multiple actions when policy has ALL_ACTION", () => {
      const policies = [policy([id("channel", "1")], [access.ALL_ACTION])];
      const allowed = access.allowRequest(
        {
          subject: id("user", "u1"),
          actions: [access.RETRIEVE_ACTION, access.DELETE_ACTION, access.CREATE_ACTION],
          objects: id("channel", "1"),
        },
        policies,
      );
      expect(allowed).toBe(true);
    });
  });

  describe("multiple objects", () => {
    it("should allow when all objects are covered", () => {
      const policies = [
        policy([id("channel", "1"), id("channel", "2")], [access.RETRIEVE_ACTION]),
      ];
      const allowed = access.allowRequest(
        {
          subject: id("user", "u1"),
          actions: access.RETRIEVE_ACTION,
          objects: [id("channel", "1"), id("channel", "2")],
        },
        policies,
      );
      expect(allowed).toBe(true);
    });

    it("should deny when some objects are not covered", () => {
      const policies = [policy([id("channel", "1")], [access.RETRIEVE_ACTION])];
      const allowed = access.allowRequest(
        {
          subject: id("user", "u1"),
          actions: access.RETRIEVE_ACTION,
          objects: [id("channel", "1"), id("channel", "2")],
        },
        policies,
      );
      expect(allowed).toBe(false);
    });

    it("should allow all objects with type-level policy", () => {
      const policies = [policy([id("channel", "")], [access.RETRIEVE_ACTION])];
      const allowed = access.allowRequest(
        {
          subject: id("user", "u1"),
          actions: access.RETRIEVE_ACTION,
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
        policy([id("channel", "1")], [access.RETRIEVE_ACTION]),
        policy([id("channel", "2")], [access.RETRIEVE_ACTION]),
      ];
      const allowed = access.allowRequest(
        {
          subject: id("user", "u1"),
          actions: access.RETRIEVE_ACTION,
          objects: [id("channel", "1"), id("channel", "2")],
        },
        policies,
      );
      expect(allowed).toBe(true);
    });

    it("should allow when one policy covers object and another has different action", () => {
      const policies = [
        policy([id("channel", "1")], [access.DELETE_ACTION]),
        policy([id("channel", "1")], [access.RETRIEVE_ACTION]),
      ];
      const allowed = access.allowRequest(
        {
          subject: id("user", "u1"),
          actions: access.RETRIEVE_ACTION,
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
          actions: access.RETRIEVE_ACTION,
          objects: id("channel", "1"),
        },
        [],
      );
      expect(allowed).toBe(false);
    });

    it("should allow with empty objects", () => {
      const policies = [policy([id("channel", "1")], [access.RETRIEVE_ACTION])];
      const allowed = access.allowRequest(
        { subject: id("user", "u1"), actions: access.RETRIEVE_ACTION, objects: [] },
        policies,
      );
      expect(allowed).toBe(true);
    });
  });
});
