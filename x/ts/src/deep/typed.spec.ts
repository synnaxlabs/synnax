// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, expectTypeOf, it } from "vitest";

import { deep } from "@/deep";

interface Scale {
  slope: number;
  units: string;
}

type Channel =
  | { type: "voltage"; key: string; port: number; scale?: Scale }
  | { type: "builtin"; key: string };

interface Values {
  name: string;
  mode: "a" | "b";
  scale?: Scale;
  channels: Channel[];
  labels: Record<string, { color: string }>;
  blob: unknown;
}

const read = <V>(_: deep.Target<V>): V => undefined as V;

describe("typed paths", () => {
  const path = deep.path<Values>();

  describe("path", () => {
    it("should stringify a root as an empty path", () => {
      expect(String(path)).toEqual("");
    });

    it("should stringify a field as a dot-separated path", () => {
      expect(String(path.name)).toEqual("name");
      expect(String(path.channels.abc.key)).toEqual("channels.abc.key");
    });

    it("should stringify a list index", () => {
      expect(String(path.channels[0].key)).toEqual("channels.0.key");
    });

    it("should return the same object for the same field", () => {
      expect(path.channels.abc.key).toBe(path.channels.abc.key);
    });

    it("should return another object for another root", () => {
      expect(deep.path<Values>().name).not.toBe(path.name);
    });

    it("should stay safe to await when a schema has a then field", async () => {
      const root = deep.path<{ then: number }>();
      expect(await Promise.resolve(root.then)).toBe(root.then);
    });

    it("should keep any field name a path", () => {
      const root = deep.path<Record<string, string>>();
      const names = ["toString", "constructor"];
      names.forEach((name) => expect(String(root[name])).toEqual(name));
    });

    it("should type a field as its value", () => {
      expectTypeOf(read(path.name)).toEqualTypeOf<string>();
      expectTypeOf(read(path.mode)).toEqualTypeOf<"a" | "b">();
      expectTypeOf(read(path.scale)).toEqualTypeOf<Scale | undefined>();
      expectTypeOf(read(path.channels)).toEqualTypeOf<Channel[]>();
      expectTypeOf(read(path.labels.abc.color)).toEqualTypeOf<string>();
      expectTypeOf(read(path.blob.deeper.still)).toEqualTypeOf<unknown>();
    });

    it("should refuse a field that the value does not have", () => {
      // @ts-expect-error a field the value does not have.
      expect(String(path.nam)).toEqual("nam");
    });

    it("should refuse a field that only some union members hold", () => {
      // @ts-expect-error `builtin` has no port.
      expect(String(path.channels[0].port)).toEqual("channels.0.port");
    });

    it("should refuse a child of an optional value", () => {
      // @ts-expect-error scale may be absent.
      expect(String(path.scale.slope)).toEqual("scale.slope");
    });

    it("should let a path flow into a wider shape", () => {
      const named: deep.Path<{ name: string }> = path;
      expect(String(named.name)).toEqual("name");
    });

    it("should refuse a path to a shape the value does not hold", () => {
      // @ts-expect-error Values has no slope.
      const scale: deep.Path<Scale> = path;
      expect(String(scale)).toEqual("");
    });

    it("should refuse a plain string", () => {
      // @ts-expect-error a path is not a string.
      read("name");
    });

    it("should show no fields to a walker", () => {
      expect(Object.keys(path.channels)).toEqual([]);
      expect(JSON.stringify(path.name)).toEqual("{}");
    });

    it("should read nothing at a symbol key", () => {
      expect(Reflect.get(path, Symbol.iterator)).toBeUndefined();
    });
  });

  describe("variant", () => {
    it("should reach the fields of the member", () => {
      const channel = deep.variant(path.channels.abc, "type", "voltage");
      expect(String(channel.port)).toEqual("channels.abc.port");
      expectTypeOf(read(channel.port)).toEqualTypeOf<number>();
    });

    it("should reach the fields shared by a union of values", () => {
      const type = "voltage" as "voltage" | "builtin";
      const channel = deep.variant(path.channels.abc, "type", type);
      expectTypeOf(read(channel.key)).toEqualTypeOf<string>();
      // @ts-expect-error only a voltage channel has a port.
      expect(String(channel.port)).toEqual("channels.abc.port");
    });

    it("should refuse a value that no member holds", () => {
      // @ts-expect-error no channel has this type.
      expect(String(deep.variant(path.channels.abc, "type", "x"))).toEqual(
        "channels.abc",
      );
    });

    it("should return the same object", () => {
      const channel = path.channels.abc;
      expect(deep.variant(channel, "type", "builtin")).toBe(channel);
    });
  });

  describe("present", () => {
    it("should reach the fields of an optional value", () => {
      const scale = deep.present(path.scale);
      expect(String(scale.slope)).toEqual("scale.slope");
      expectTypeOf(read(scale.slope)).toEqualTypeOf<number>();
    });

    it("should return the same object", () => {
      expect(deep.present(path.scale)).toBe(path.scale);
    });
  });

  describe("widen", () => {
    const keys = <C extends { key: string }>(list: deep.Path<C[]>, key: string) =>
      deep.widen<{ key: string }[]>(list)[key].key;

    it("should read a generic list through its known shape", () => {
      expect(String(keys(path.channels, "abc"))).toEqual("channels.abc.key");
    });

    it("should return the same object", () => {
      expect(deep.widen(path.name)).toBe(path.name);
    });

    it("should refuse a shape that the value does not hold", () => {
      // @ts-expect-error a string has no port.
      expect(String(deep.widen<{ port: number }>(path.name))).toEqual("name");
    });
  });
});
