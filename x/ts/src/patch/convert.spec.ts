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

import { caseconv } from "@/caseconv";
import { patch } from "@/patch";

const channelZ = z.discriminatedUnion("type", [
  z.object({ key: z.string(), type: z.literal("ai_voltage"), maxVal: z.number() }),
  z.object({ key: z.string(), type: z.literal("ai_bridge"), bridgeConfig: z.string() }),
]);

const schema = z.object({
  sampleRate: z.number(),
  channels: z.array(channelZ),
  outputChannels: z.array(channelZ),
  nodeProps: caseconv.preserveKeys(
    z.record(z.string(), z.object({ fillColor: z.string() })),
  ),
});

const doc = {
  sampleRate: 1,
  channels: [{ key: "ch_a", type: "ai_voltage", maxVal: 10 }],
  outputChannels: [{ key: "ch_b", type: "ai_voltage", maxVal: 5 }],
  nodeProps: { node_a: { fillColor: "red" } },
};

const opts = { schema, data: doc };

describe("convert", () => {
  describe("camelToSnake", () => {
    it("should convert the path and value of a set", () => {
      expect(
        patch.camelToSnake(
          {
            type: "set",
            set: {
              path: ["channels", "ch_a"],
              value: { key: "ch_a", type: "ai_bridge", bridgeConfig: "full" },
              guards: [],
            },
          },
          opts,
        ),
      ).toEqual({
        type: "set",
        set: {
          path: ["channels", "ch_a"],
          value: { key: "ch_a", type: "ai_bridge", bridge_config: "full" },
          guards: [],
        },
      });
    });

    it("should convert the paths of the guards", () => {
      expect(
        patch.camelToSnake(
          {
            type: "set",
            set: {
              path: ["outputChannels", "ch_b", "maxVal"],
              value: 1,
              guards: [
                { path: ["outputChannels", "ch_b", "type"], value: "ai_voltage" },
              ],
            },
          },
          opts,
        ),
      ).toEqual({
        type: "set",
        set: {
          path: ["output_channels", "ch_b", "max_val"],
          value: 1,
          guards: [{ path: ["output_channels", "ch_b", "type"], value: "ai_voltage" }],
        },
      });
    });

    it("should keep the keys of a record with preserved keys", () => {
      const converted = patch.camelToSnake(
        {
          type: "set",
          set: {
            path: ["nodeProps", "node_a", "fillColor"],
            value: "blue",
            guards: [],
          },
        },
        opts,
      );
      expect(converted).toEqual({
        type: "set",
        set: {
          path: ["node_props", "node_a", "fill_color"],
          value: "blue",
          guards: [],
        },
      });
    });

    it("should convert the key of a remove from an object", () => {
      expect(
        patch.camelToSnake(
          { type: "remove", remove: { path: [], key: "sampleRate", guards: [] } },
          opts,
        ),
      ).toEqual({
        type: "remove",
        remove: { path: [], key: "sample_rate", guards: [] },
      });
    });

    it("should keep the key of a remove from a list", () => {
      expect(
        patch.camelToSnake(
          { type: "remove", remove: { path: ["channels"], key: "ch_a", guards: [] } },
          opts,
        ),
      ).toEqual({
        type: "remove",
        remove: { path: ["channels"], key: "ch_a", guards: [] },
      });
    });

    it("should convert an inserted entry with the element schema", () => {
      expect(
        patch.camelToSnake(
          {
            type: "insert",
            insert: {
              path: ["channels"],
              value: { key: "ch_b", type: "ai_voltage", maxVal: 5 },
              after: "ch_a",
              guards: [],
            },
          },
          opts,
        ),
      ).toEqual({
        type: "insert",
        insert: {
          path: ["channels"],
          value: { key: "ch_b", type: "ai_voltage", max_val: 5 },
          after: "ch_a",
          guards: [],
        },
      });
    });

    it("should keep the keys of a move", () => {
      expect(
        patch.camelToSnake(
          {
            type: "move",
            move: { path: ["channels"], key: "ch_a", after: "ch_b", guards: [] },
          },
          opts,
        ),
      ).toEqual({
        type: "move",
        move: { path: ["channels"], key: "ch_a", after: "ch_b", guards: [] },
      });
    });
  });

  describe("snakeToCamel", () => {
    it("should invert camelToSnake", () => {
      const action: patch.Action = {
        type: "set",
        set: {
          path: ["outputChannels", "ch_b", "maxVal"],
          value: 3,
          guards: [{ path: ["outputChannels", "ch_b", "type"], value: "ai_voltage" }],
        },
      };
      expect(patch.snakeToCamel(patch.camelToSnake(action, opts), opts)).toEqual(
        action,
      );
    });

    it("should convert a value applied to the document as the codec would", () => {
      const wire = caseconv.camelToSnake(doc, { schema });
      const action = patch.snakeToCamel(
        {
          type: "set",
          set: { path: ["channels", "ch_a"], value: wire.channels[0], guards: [] },
        },
        opts,
      );
      const next = structuredClone(doc);
      patch.apply(next, action);
      expect(next).toEqual(doc);
    });
  });

  describe("guards", () => {
    it("should build a guard for each union the path goes into", () => {
      expect(patch.guards(schema, ["channels", "ch_a", "maxVal"], doc)).toEqual([
        { path: ["channels", "ch_a", "type"], value: "ai_voltage" },
      ]);
    });

    it("should throw for a discriminator that is not a string", () => {
      const numbered = z.object({
        v: z.discriminatedUnion("n", [
          z.object({ n: z.literal(1), a: z.number() }),
          z.object({ n: z.literal(2), b: z.number() }),
        ]),
      });
      expect(() => patch.guards(numbered, ["v", "a"], { v: { n: 1, a: 0 } })).toThrow(
        "The discriminator at v.n is not a string",
      );
    });
  });
});
