// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { ranger } from "@/ranger";

describe("alias", () => {
  describe("ontologyID", () => {
    it("should create an ontology ID with the correct type and composite key", () => {
      const id = ranger.alias.ontologyID("range-uuid-123", 42);
      expect(id.type).toBe("range-alias");
      expect(id.key).toBe("range-uuid-123---42");
    });

    it("should match the key format produced by createKey", () => {
      const rangeKey = "abc-def";
      const channelKey = 7;
      const id = ranger.alias.ontologyID(rangeKey, channelKey);
      const key = ranger.alias.createKey({ range: rangeKey, channel: channelKey });
      expect(id.key).toBe(key);
    });
  });

  describe("TYPE_ONTOLOGY_ID", () => {
    it("should have the correct type and an empty key", () => {
      expect(ranger.alias.TYPE_ONTOLOGY_ID.type).toBe("range-alias");
      expect(ranger.alias.TYPE_ONTOLOGY_ID.key).toBe("");
    });
  });

  describe("createKey", () => {
    it("should create a composite key from range and channel", () => {
      const key = ranger.alias.createKey({ range: "my-range", channel: 10 });
      expect(key).toBe("my-range---10");
    });
  });

  describe("decodeDeleteChange", () => {
    it("should decode a composite key into range and channel", () => {
      const decoded = ranger.alias.decodeDeleteChange("my-range---10");
      expect(decoded.range).toBe("my-range");
      expect(decoded.channel).toBe(10);
    });
  });
});
