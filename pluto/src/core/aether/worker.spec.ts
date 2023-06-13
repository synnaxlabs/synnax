// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { AtherComposite, AetherFactory, AetherLeaf } from "@/core/aether/worker";

export const exampleProps = z.object({
  x: z.number(),
});

type ExampleProps = z.input<typeof exampleProps>;
type ParsedExampleProps = z.output<typeof exampleProps>;

export class ExampleFactory
  implements AetherFactory<AetherLeaf<ExampleProps, ParsedExampleProps>>
{
  create(
    type: string,
    key: string,
    state: ExampleProps
  ): AetherLeaf<ExampleProps, ParsedExampleProps> {
    return new AetherLeaf(type, key, state, exampleProps);
  }
}

describe("Bob Worker", () => {
  describe("BobLeaf", () => {
    describe("setState", () => {
      it("should throw an error if the path is empty", () => {
        const leaf = new AetherLeaf("test", "test", { x: 1 }, exampleProps);
        expect(() => leaf.setState([], "test", { x: 1 })).toThrowError();
      });
      it("should throw an error if the path has a subpath", () => {
        const leaf = new AetherLeaf("test", "test", { x: 1 }, exampleProps);
        expect(() => leaf.setState(["test", "dog"], "test", { x: 1 })).toThrowError();
      });
      it("should throw an error if the path does not have the correct key", () => {
        const leaf = new AetherLeaf("test", "test", { x: 1 }, exampleProps);
        expect(() => leaf.setState(["dog"], "test", { x: 1 })).toThrowError();
      });
      it("should correctly set the state", () => {
        const leaf = new AetherLeaf("test", "test", { x: 1 }, exampleProps);
        leaf.setState(["test"], "test", { x: 2 });
        expect(leaf.state).toEqual({ x: 2 });
      });
      it("should call the state hook", () => {
        const leaf = new AetherLeaf("test", "test", { x: 1 }, exampleProps);
        const called = vi.fn();
        leaf.setStateHook(called);
        leaf.setState(["test"], "test", { x: 2 });
        expect(called).toHaveBeenCalled();
      });
    });
    describe("delete", () => {
      it("should call the delete hook", () => {
        const leaf = new AetherLeaf("test", "test", { x: 1 }, exampleProps);
        const called = vi.fn();
        leaf.setDeleteHook(called);
        leaf.delete(["test"]);
        expect(called).toHaveBeenCalled();
      });
    });
  });
  describe("BobComposite", () => {
    describe("setState", () => {
      it("should set the state of the composite's leaf if the path has one element", () => {
        const composite = new AtherComposite(
          "test",
          "test",
          new ExampleFactory(),
          exampleProps,
          { x: 1 }
        );
        composite.setState(["test"], "test", { x: 2 });
      });
      it("should create a new leaf if the path has more than one element and the leaf does not exist", () => {
        const composite = new AtherComposite(
          "test",
          "test",
          new ExampleFactory(),
          exampleProps,
          { x: 1 }
        );
        composite.setState(["test", "dog"], "test", { x: 2 });
        expect(composite.children).toHaveLength(1);
      });
      it("should set the state of the composite's leaf if the path has more than one element and the leaf exists", () => {
        const composite = new AtherComposite(
          "test",
          "test",
          new ExampleFactory(),
          exampleProps,
          { x: 1 }
        );
        composite.setState(["test", "dog"], "test", { x: 2 });
        composite.setState(["test", "dog"], "test", { x: 3 });
        expect(composite.children).toHaveLength(1);
        expect(composite.children[0].state).toEqual({ x: 3 });
      });
      it("should throw an error if the path is too deep and the child does not exist", () => {
        const composite = new AtherComposite(
          "test",
          "test",
          new ExampleFactory(),
          exampleProps,
          { x: 1 }
        );
        expect(() =>
          composite.setState(["test", "dog", "cat"], "test", { x: 2 })
        ).toThrowError();
      });
    });
    describe("delete", () => {
      it("should remove a child from the list of children", () => {
        const composite = new AtherComposite(
          "test",
          "test",
          new ExampleFactory(),
          exampleProps,
          { x: 1 }
        );
        composite.setState(["test", "dog"], "test", { x: 2 });
        composite.delete(["test", "dog"]);
        expect(composite.children).toHaveLength(0);
      });
      it("should call the deletion hook on the child of a composite", () => {
        const composite = new AtherComposite(
          "test",
          "test",
          new ExampleFactory(),
          exampleProps,
          { x: 1 }
        );
        composite.setState(["test", "dog"], "test", { x: 2 });
        const called = vi.fn();
        composite.children[0].setDeleteHook(called);
        composite.delete(["test", "dog"]);
        expect(called).toHaveBeenCalled();
      });
    });
  });
});
