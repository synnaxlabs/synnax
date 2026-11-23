// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, icd } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { ICD } from "@/platform/icd";

describe("ICD.Stored", () => {
  describe("bindingsOf", () => {
    const [a, a2, b, c, g, i] = Array.from({ length: 6 }, () => uuid.create());
    const dataTypes = new Map<channel.Key, string>([
      [1, "float64"],
      [2, "string"],
      [4, "int32"],
    ]);
    const channelsOf = (
      fields: {
        key: string;
        pointer: string;
        type: "float" | "integer" | "string";
        scale?: number;
      }[],
      written = new Set<channel.Key>(),
    ) =>
      ICD.Stored.bindingsOf(
        { "/a": 1, "/b": 2, "/gone": 3, "/i": 4 },
        fields.map((f) => icd.jsonFieldZ.parse({ name: "field", ...f })),
        written,
        dataTypes,
      ).map(({ field, channel }) => [field, channel]);

    it("should bind each field to the channel stored for its pointer", () => {
      const written = new Set<channel.Key>();
      expect(
        channelsOf(
          [
            { key: a, pointer: "/a", type: "float" },
            { key: b, pointer: "/b", type: "string" },
          ],
          written,
        ),
      ).toEqual([
        [a, 1],
        [b, 2],
      ]);
      expect(written).toEqual(new Set([1, 2]));
    });

    it("should bind no channel for a pointer with no stored channel", () => {
      expect(channelsOf([{ key: c, pointer: "/c", type: "float" }])).toEqual([[c, 0]]);
    });

    it("should bind no channel that another binding writes", () => {
      expect(
        channelsOf(
          [
            { key: a, pointer: "/a", type: "float" },
            { key: a2, pointer: "/a", type: "float" },
          ],
          new Set(),
        ),
      ).toEqual([
        [a, 1],
        [a2, 0],
      ]);
      expect(
        channelsOf([{ key: a, pointer: "/a", type: "float" }], new Set([1])),
      ).toEqual([[a, 0]]);
    });

    it("should bind no channel whose data type the kind does not convert to", () => {
      expect(channelsOf([{ key: a, pointer: "/a", type: "string" }])).toEqual([[a, 0]]);
    });

    it("should bind a scaled field to a float channel only", () => {
      expect(
        channelsOf([
          { key: a, pointer: "/a", type: "integer", scale: 2 },
          { key: i, pointer: "/i", type: "integer", scale: 2 },
        ]),
      ).toEqual([
        [a, 1],
        [i, 0],
      ]);
    });

    it("should bind no channel whose data type is unknown", () => {
      expect(channelsOf([{ key: g, pointer: "/gone", type: "float" }])).toEqual([
        [g, 0],
      ]);
    });
  });

  describe("sources", () => {
    const [m1, m2] = [uuid.create(), uuid.create()];
    const props = (index: channel.Key, channels: Record<string, channel.Key>) => ({
      index,
      channels,
    });
    const read = (message: string, bound: channel.Key[] = []) =>
      icd.readMessageZ.parse({
        message,
        bindings: bound.map((channel) => ({ field: uuid.create(), channel })),
      });
    const source = (
      messages: icd.ReadMessage[],
      legacy?: string,
    ): ICD.Stored.Source => ({ key: "t", legacy, messages });

    describe("of", () => {
      it("should return the channels stored under the key of the message", () => {
        const own = props(1, { "/a": 2 });
        expect(
          ICD.Stored.of(
            { [ICD.Stored.key("t", m1)]: own },
            source([read(m1, [2])], "t"),
            0,
          ),
        ).toBe(own);
      });

      it("should give the legacy channels to the message that binds one", () => {
        const held = props(1, { "/a": 2 });
        const messages = [read(m1), read(m2, [2])];
        expect(ICD.Stored.of({ t: held }, source(messages, "t"), 0)).toBeUndefined();
        expect(ICD.Stored.of({ t: held }, source(messages, "t"), 1)).toBe(held);
      });

      it("should give the legacy channels to the first message by default", () => {
        const held = props(1, { "/a": 2 });
        const messages = [read(m1), read(m2)];
        expect(ICD.Stored.of({ t: held }, source(messages, "t"), 0)).toBe(held);
        expect(ICD.Stored.of({ t: held }, source(messages, "t"), 1)).toBeUndefined();
      });

      it("should read no legacy channels for a source with no legacy key", () => {
        expect(
          ICD.Stored.of({ t: props(1, { "/a": 2 }) }, source([read(m1)]), 0),
        ).toBeUndefined();
      });
    });

    describe("channels", () => {
      it("should merge the channels of the source, with the message first", () => {
        expect(
          ICD.Stored.channels(
            {
              t: props(1, { "/a": 2, "/b": 3 }),
              [ICD.Stored.key("t", m2)]: props(1, { "/b": 4, "/c": 5 }),
              [ICD.Stored.key("t", m1)]: props(1, { "/c": 6 }),
              [ICD.Stored.key("u", m1)]: props(1, { "/d": 7 }),
            },
            source([read(m1)], "t"),
            m1,
          ),
        ).toEqual({ "/a": 2, "/b": 4, "/c": 6 });
      });

      it("should leave the channels of a later read with that read", () => {
        expect(
          ICD.Stored.channels(
            {
              [ICD.Stored.key("t", m1)]: props(1, { "/a": 2 }),
              [ICD.Stored.key("t", m2)]: props(1, { "/b": 3 }),
            },
            source([read(m1), read(m2)]),
            m1,
          ),
        ).toEqual({ "/a": 2 });
      });

      it("should merge no bare key for a source with no legacy key", () => {
        expect(
          ICD.Stored.channels({ t: props(1, { "/a": 2 }) }, source([read(m1)]), m1),
        ).toEqual({});
      });

      it("should merge no channels of a topic that extends the topic", () => {
        expect(
          ICD.Stored.channels(
            {
              [ICD.Stored.key("a/", m1)]: props(1, { "/a": 2 }),
              "a/#": props(1, { "/b": 3 }),
              [ICD.Stored.key("a/#", m2)]: props(1, { "/c": 4 }),
            },
            { key: "a/", legacy: "a/", messages: [read(m1)] },
            m1,
          ),
        ).toEqual({ "/a": 2 });
      });

      it("should merge no channels of a record whose key extends the key", () => {
        expect(
          ICD.Stored.channels(
            {
              [ICD.Stored.key("t#a", m1)]: props(1, { "/a": 2 }),
              [ICD.Stored.key("t#a#b", m2)]: props(1, { "/b": 3 }),
            },
            { key: "t#a", messages: [read(m1)] },
            m1,
          ),
        ).toEqual({ "/a": 2 });
      });
    });

    describe("moveLegacy", () => {
      it("should move the legacy channels to the key of their read", () => {
        const held = props(1, { "/a": 2 });
        const stored: Record<string, ReturnType<typeof props>> = { t: held };
        ICD.Stored.moveLegacy(stored, source([read(m1), read(m2, [2])], "t"), 0);
        expect(stored).toEqual({ t: held });
        ICD.Stored.moveLegacy(stored, source([read(m1), read(m2, [2])], "t"), 1);
        expect(stored).toEqual({ [ICD.Stored.key("t", m2)]: held });
      });

      it("should keep the legacy channels when the read has its own", () => {
        const held = props(1, { "/a": 2 });
        const own = props(1, { "/b": 3 });
        const stored = { t: held, [ICD.Stored.key("t", m1)]: own };
        ICD.Stored.moveLegacy(stored, source([read(m1)], "t"), 0);
        expect(stored).toEqual({ t: held, [ICD.Stored.key("t", m1)]: own });
      });
    });
  });
});
