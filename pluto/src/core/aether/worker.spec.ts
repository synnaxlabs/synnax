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
  AetherUpdate,
  AetherComposite,
  AetherContext,
  AetherLeaf,
  AetherComponentRegistry,
} from "@/core/aether/worker";

export const exampleProps = z.object({
  x: z.number(),
});

interface ExampleLeafDerived {
  contextValue: number;
}

class ExampleLeaf extends AetherLeaf<typeof exampleProps, ExampleLeafDerived> {
  updatef = vi.fn();
  deletef = vi.fn();
  schema = exampleProps;

  derive(): ExampleLeafDerived {
    this.updatef(this.ctx);
    return { contextValue: this.ctx.getOptional("key") ?? 0 };
  }

  afterDelete(): void {
    this.deletef();
  }
}

class ExampleComposite extends AetherComposite<
  typeof exampleProps,
  void,
  ExampleLeaf | ContextSetterComposite
> {
  updatef = vi.fn();
  deletef = vi.fn();

  schema = exampleProps;

  derive(): void {
    this.updatef(this.ctx);
  }

  afterDelete(): void {
    this.deletef();
  }
}

class ContextSetterComposite extends AetherComposite<
  typeof exampleProps,
  void,
  ExampleLeaf
> {
  updatef = vi.fn();
  deletef = vi.fn();

  schema = exampleProps;

  derive(): void {
    this.updatef(this.ctx);
    this.ctx.set("key", this.state.x);
  }

  afterDelete(): void {
    this.deletef();
  }
}

const REGISTRY: AetherComponentRegistry = {
  leaf: (internalUpdate: AetherUpdate) => new ExampleLeaf(internalUpdate),
  composite: (internalUpdate: AetherUpdate) => new ExampleComposite(internalUpdate),
  context: (interalUpdate: AetherUpdate) => new ContextSetterComposite(interalUpdate),
};

const MockSender = {
  send: vi.fn(),
};

const ctx = new AetherContext(MockSender, REGISTRY);

const leafUpdate: AetherUpdate = {
  ctx,
  variant: "state",
  type: "leaf",
  path: ["test"],
  state: { x: 1 },
};

const compositeUpdate: AetherUpdate = {
  ctx,
  variant: "state",
  type: "composite",
  path: ["test"],
  state: { x: 1 },
};

const contextUpdate: AetherUpdate = {
  ctx,
  variant: "context",
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
    describe("internalUpdate", () => {
      it("should throw an error if the path is empty", () => {
        expect(() => leaf.internalUpdate({ ...leafUpdate, path: [] })).toThrowError(
          /empty path/
        );
      });
      it("should throw an error if the path has a subpath", () => {
        expect(() =>
          leaf.internalUpdate({ ...leafUpdate, path: ["test", "dog"] })
        ).toThrowError(/subPath/);
      });
      it("should throw an error if the path does not have the correct key", () => {
        expect(() =>
          leaf.internalUpdate({ ...leafUpdate, path: ["dog"] })
        ).toThrowError(/key/);
      });
      it("should correctly internalUpdate the state", () => {
        leaf.internalUpdate({ ...leafUpdate, state: { x: 2 } });
        expect(leaf.state).toEqual({ x: 2 });
      });
      it("should call the handleUpdate", () => {
        leaf.internalUpdate({ ...leafUpdate, state: { x: 2 } });
        expect(leaf.updatef).toHaveBeenCalledTimes(2);
      });
    });
    describe("internalDelete", () => {
      it("should call the bound onDelete handler", () => {
        leaf.internalDelete(["test"]);
        expect(leaf.deletef).toHaveBeenCalledTimes(1);
      });
    });
    describe("setState", () => {
      it("should communicate the state call to the main thread Sender", () => {
        leaf.setState((p) => ({ ...p }));
        expect(MockSender.send).toHaveBeenCalledTimes(1);
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
        composite.internalUpdate({ ...compositeUpdate, state: { x: 2 } });
      });
      it("should create a new leaf if the path has more than one element and the leaf does not exist", () => {
        composite.internalUpdate({
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
        composite.internalUpdate({
          ...leafUpdate,
          path: ["test", "dog"],
          state: { x: 2 },
        });
        composite.internalUpdate({
          ...leafUpdate,
          path: ["test", "dog"],
          state: { x: 3 },
        });
        expect(composite.children).toHaveLength(1);
        expect(composite.children[0].state).toEqual({ x: 3 });
      });
      it("should throw an error if the path is too deep and the child does not exist", () => {
        expect(() =>
          composite.internalUpdate({ ...compositeUpdate, path: ["test", "dog", "cat"] })
        ).toThrowError();
      });
    });
    describe("internalDelete", () => {
      it("should remove a child from the list of children", () => {
        composite.internalUpdate({ ...compositeUpdate, path: ["test", "dog"] });
        composite.internalDelete(["test", "dog"]);
        expect(composite.children).toHaveLength(0);
      });
      it("should call the deletion hook on the child of a composite", () => {
        composite.internalUpdate({ ...leafUpdate, path: ["test", "dog"] });
        const c = composite.children[0];
        composite.internalDelete(["test", "dog"]);
        expect(c.deletef).toHaveBeenCalled();
      });
    });

    describe("context propagation", () => {
      it("should properly propagate an existing context change to its children", () => {
        composite.internalUpdate({ ...leafUpdate, path: ["test", "dog"] });
        composite.internalUpdate({ ...leafUpdate, path: ["test", "cat"] });
        expect(composite.children).toHaveLength(2);
        composite.internalUpdate({ ...contextUpdate });
        expect(composite.updatef).toHaveBeenCalledTimes(2);
        composite.children.forEach((c) => expect(c.updatef).toHaveBeenCalledTimes(2));
      });
      it.only("should progate a new context change to its children", () => {
        const c = new ContextSetterComposite({ ...compositeUpdate });
        c.internalUpdate({ ...leafUpdate, path: ["test", "dog"] });
        c.internalUpdate({ ...compositeUpdate });
        expect(c.children).toHaveLength(1);
        c.children.forEach((c) => expect(c.updatef).toHaveBeenCalledTimes(2));
      });
      it("should correctly separate individual contexts", () => {
        composite.internalUpdate({
          ctx,
          variant: "state",
          type: "context",
          path: ["test", "dog"],
          state: { x: 1 },
        });
        composite.internalUpdate({
          ctx,
          variant: "state",
          type: "context",
          path: ["test", "cat"],
          state: { x: 2 },
        });
        expect(composite.children).toHaveLength(2);
        composite.internalUpdate({
          ctx,
          variant: "state",
          type: "leaf",
          path: ["test", "dog", "dogleaf"],
          state: { x: 3 },
        });
        composite.internalUpdate({
          ctx,
          variant: "state",
          type: "leaf",
          path: ["test", "cat", "catLeaf"],
          state: { x: 4 },
        });
        composite.internalUpdate({
          ctx,
          variant: "state",
          type: "context",
          path: ["test", "dog"],
          state: { x: 5 },
        });
      });
    });
  });
});
