// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import {
  Update,
  AetherComposite,
  AetherContext,
  AetherLeaf,
} from "@/core/aether/worker";

export const exampleProps = z.object({
  x: z.number(),
});

class ExampleLeaf extends AetherLeaf<typeof exampleProps> {
  updatef = vi.fn();
  deletef = vi.fn();

  handleUpdate(ctx: AetherContext): void {
    this.updatef(ctx);
  }

  handleDelete(): void {
    this.deletef();
  }
}

const ctx = new AetherContext({
  example: (update: Update) => new ExampleLeaf(update, exampleProps),
});

const update: Update = {
  ctx,
  type: "example",
  path: ["test"],
  state: { x: 1 },
};

describe("Aether Worker", () => {
  describe("AetherLeaf", () => {
    let leaf: ExampleLeaf;
    beforeEach(() => {
      leaf = ctx.create(update);
    });
    describe("update", () => {
      it("should throw an error if the path is empty", () => {
        expect(() => leaf.update({ ...update, path: [] })).toThrowError();
      });
      it("should throw an error if the path has a subpath", () => {
        expect(() => leaf.update({ ...update, path: ["test", "dog"] })).toThrowError();
      });
      it("should throw an error if the path does not have the correct key", () => {
        expect(() => leaf.update({ ...update, path: ["dog"] })).toThrowError();
      });
      it("should correctly update the state", () => {
        leaf.update({ ...update, state: { x: 2 } });
        expect(leaf.state).toEqual({ x: 2 });
      });
      it("should call the handleUpdate", () => {
        leaf.update({ ...update, state: { x: 2 } });
        expect(leaf.updatef).toHaveBeenCalledTimes(1);
      });
    });
    describe("delete", () => {
      it("should call the bound onDelete handler", () => {
        leaf.delete(["test"]);
        expect(leaf.deletef).toHaveBeenCalledTimes(1);
      });
    });
  });
  describe("AetherComposite", () => {
    describe("setState", () => {
      it("should set the state of the composite's leaf if the path has one element", () => {
        const composite = new AetherComposite(
          "test",
          "test",
          new ExampleFactory(),
          exampleProps,
          { x: 1 }
        );
        composite.update(["test"], "test", { x: 2 });
      });
      it("should create a new leaf if the path has more than one element and the leaf does not exist", () => {
        const composite = new AetherComposite(
          "test",
          "test",
          new ExampleFactory(),
          exampleProps,
          { x: 1 }
        );
        composite.update(["test", "dog"], "test", { x: 2 });
        expect(composite.children).toHaveLength(1);
      });
      it("should set the state of the composite's leaf if the path has more than one element and the leaf exists", () => {
        const composite = new AetherComposite(
          "test",
          "test",
          new ExampleFactory(),
          exampleProps,
          { x: 1 }
        );
        composite.update(["test", "dog"], "test", { x: 2 });
        composite.update(["test", "dog"], "test", { x: 3 });
        expect(composite.children).toHaveLength(1);
        expect(composite.children[0].state).toEqual({ x: 3 });
      });
      it("should throw an error if the path is too deep and the child does not exist", () => {
        const composite = new AetherComposite(
          "test",
          "test",
          new ExampleFactory(),
          exampleProps,
          { x: 1 }
        );
        expect(() =>
          composite.update(["test", "dog", "cat"], "test", { x: 2 })
        ).toThrowError();
      });
    });
    describe("delete", () => {
      it("should remove a child from the list of children", () => {
        const composite = new AetherComposite(
          "test",
          "test",
          new ExampleFactory(),
          exampleProps,
          { x: 1 }
        );
        composite.update(["test", "dog"], "test", { x: 2 });
        composite.delete(["test", "dog"]);
        expect(composite.children).toHaveLength(0);
      });
      it("should call the deletion hook on the child of a composite", () => {
        const composite = new AetherComposite(
          "test",
          "test",
          new ExampleFactory(),
          exampleProps,
          { x: 1 }
        );
        composite.update(["test", "dog"], "test", { x: 2 });
        const called = vi.fn();
        composite.children[0].setDeleteHook(called);
        composite.delete(["test", "dog"]);
        expect(called).toHaveBeenCalled();
      });
    });
  });
});
