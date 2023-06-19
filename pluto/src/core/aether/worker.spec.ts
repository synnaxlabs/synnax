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

  constructor(update: Update) {
    super(update, exampleProps);
  }

  handleUpdate(ctx: AetherContext): void {
    this.updatef(ctx);
  }

  handleDelete(): void {
    this.deletef();
  }
}

class ExampleComposite extends AetherComposite<typeof exampleProps, ExampleLeaf> {
  updatef = vi.fn();
  deletef = vi.fn();

  constructor(update: Update) {
    super(update, exampleProps);
  }

  handleUpdate(ctx: AetherContext): void {
    this.updatef(ctx);
  }

  handleDelete(): void {
    this.deletef();
  }
}

class ContextSetterComposite extends AetherComposite<typeof exampleProps, ExampleLeaf> {
  updatef = vi.fn();
  deletef = vi.fn();

  constructor(update: Update) {
    super(update, exampleProps);
  }

  handleUpdate(ctx: AetherContext): void {
    this.updatef(ctx);
    ctx.set("key", "value");
  }

  handleDelete(): void {
    this.deletef();
  }
}

const ctx = new AetherContext({
  leaf: (update: Update) => new ExampleLeaf(update),
  composite: (update: Update) => new ExampleComposite(update),
});

const leafUpdate: Update = {
  ctx,
  type: "leaf",
  path: ["test"],
  state: { x: 1 },
};

const compositeUpdate: Update = {
  ctx,
  type: "composite",
  path: ["test"],
  state: { x: 1 },
};

const contextUpdate: Update = {
  ctx,
  type: "context",
  path: [],
  state: null,
};

describe("Aether Worker", () => {
  describe("AetherLeaf", () => {
    let leaf: ExampleLeaf;
    beforeEach(() => {
      leaf = ctx.create(leafUpdate);
    });
    describe("update", () => {
      it("should throw an error if the path is empty", () => {
        expect(() => leaf.update({ ...leafUpdate, path: [] })).toThrowError(
          /empty path/
        );
      });
      it("should throw an error if the path has a subpath", () => {
        expect(() =>
          leaf.update({ ...leafUpdate, path: ["test", "dog"] })
        ).toThrowError(/subPath/);
      });
      it("should throw an error if the path does not have the correct key", () => {
        expect(() => leaf.update({ ...leafUpdate, path: ["dog"] })).toThrowError(/key/);
      });
      it("should correctly update the state", () => {
        leaf.update({ ...leafUpdate, state: { x: 2 } });
        expect(leaf.state).toEqual({ x: 2 });
      });
      it("should call the handleUpdate", () => {
        leaf.update({ ...leafUpdate, state: { x: 2 } });
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
    let composite: ExampleComposite;
    beforeEach(() => {
      composite = ctx.create(compositeUpdate);
    });
    describe("setState", () => {
      it("should set the state of the composite's leaf if the path has one element", () => {
        composite.update({ ...compositeUpdate, state: { x: 2 } });
      });
      it("should create a new leaf if the path has more than one element and the leaf does not exist", () => {
        composite.update({
          ...leafUpdate,
          path: ["test", "dog"],
          state: { x: 2 },
        });
        expect(composite.children).toHaveLength(1);
        const c = composite.children[0];
        expect(c.key).toEqual("dog");
        expect(c.state).toEqual({ x: 2 });
      });
      it("should set the state of the composite's leaf if the path has more than one element and the leaf exists", () => {
        composite.update({
          ...leafUpdate,
          path: ["test", "dog"],
          state: { x: 2 },
        });
        composite.update({
          ...leafUpdate,
          path: ["test", "dog"],
          state: { x: 3 },
        });
        expect(composite.children).toHaveLength(1);
        expect(composite.children[0].state).toEqual({ x: 3 });
      });
      it("should throw an error if the path is too deep and the child does not exist", () => {
        expect(() =>
          composite.update({ ...compositeUpdate, path: ["test", "dog", "cat"] })
        ).toThrowError();
      });
    });
    describe("delete", () => {
      it("should remove a child from the list of children", () => {
        composite.update({ ...compositeUpdate, path: ["test", "dog"] });
        composite.delete(["test", "dog"]);
        expect(composite.children).toHaveLength(0);
      });
      it("should call the deletion hook on the child of a composite", () => {
        composite.update({ ...leafUpdate, path: ["test", "dog"] });
        const c = composite.children[0];
        composite.delete(["test", "dog"]);
        expect(c.deletef).toHaveBeenCalled();
      });
    });

    describe("context propagation", () => {
      it("should properly propagate an existing context change to its children", () => {
        composite.update({ ...leafUpdate, path: ["test", "dog"] });
        composite.update({ ...leafUpdate, path: ["test", "cat"] });
        expect(composite.children).toHaveLength(2);
        composite.update({ ...contextUpdate });
        expect(composite.updatef).toHaveBeenCalledTimes(1);
        composite.children.forEach((c) => expect(c.updatef).toHaveBeenCalledTimes(1));
      });
      it("should progate a new context change to its children", () => {
        const c = new ContextSetterComposite({ ...compositeUpdate });
        c.update({ ...leafUpdate, path: ["test", "dog"] });
        c.update({ ...compositeUpdate });
        expect(c.children).toHaveLength(1);
        c.children.forEach((c) => expect(c.updatef).toHaveBeenCalledTimes(1));
      });
    });
  });
});
