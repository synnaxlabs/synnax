// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { aether } from "@/aether/aether";

export const exampleProps = z.object({
  x: z.number(),
});

interface InternalState {
  contextValue: number;
}

class ExampleLeaf extends aether.Leaf<typeof exampleProps, InternalState> {
  updatef = vi.fn();
  deletef = vi.fn();
  schema = exampleProps;

  async afterUpdate(): Promise<void> {
    this.updatef(this.ctx);
    this.internal.contextValue = this.ctx.getOptional("key") ?? 0;
  }

  async afterDelete(): Promise<void> {
    this.deletef();
  }
}

class ExampleComposite extends aether.Composite<
  typeof exampleProps,
  {},
  ExampleLeaf | ContextSetterComposite
> {
  updatef = vi.fn();
  deletef = vi.fn();

  schema = exampleProps;

  async afterUpdate(): Promise<void> {
    this.updatef(this.ctx);
  }

  async afterDelete(): Promise<void> {
    this.deletef();
  }
}

class ContextSetterComposite extends aether.Composite<
  typeof exampleProps,
  {},
  ExampleLeaf | SecondaryContextSetter
> {
  updatef = vi.fn();
  deletef = vi.fn();

  schema = exampleProps;

  async afterUpdate(): Promise<void> {
    this.updatef(this.ctx);
    this.ctx.set("key", this.state.x);
  }

  async afterDelete(): Promise<void> {
    this.deletef();
  }
}
class SecondaryContextSetter extends aether.Composite<
  typeof exampleProps,
  {},
  ExampleLeaf
> {
  updatef = vi.fn();
  deletef = vi.fn();

  schema = exampleProps;

  async afterUpdate(): Promise<void> {
    this.updatef(this.ctx);
    const v = this.ctx.getOptional<number>("key");
    if (v != null) this.ctx.set("key2", v + 1);
  }
}
const REGISTRY: aether.ComponentRegistry = {
  leaf: ExampleLeaf,
  composite: ExampleComposite,
  context: ContextSetterComposite,
  secondary: SecondaryContextSetter,
};

const MockSender = {
  send: vi.fn(),
};

const ctx = new aether.Context(MockSender, REGISTRY);

const leafUpdate: aether.Update = {
  ctx,
  variant: "state",
  type: "leaf",
  path: ["test"],
  state: { x: 1 },
  instrumentation: alamos.NOOP,
};

const compositeUpdate: aether.Update = {
  ctx,
  variant: "state",
  type: "composite",
  path: ["test"],
  state: { x: 1 },
  instrumentation: alamos.NOOP,
};

const contextUpdate: aether.Update = {
  ctx,
  variant: "context",
  type: "context",
  path: [],
  state: {},
  instrumentation: alamos.NOOP,
};

describe("Aether Worker", () => {
  describe("AetherLeaf", () => {
    let leaf: ExampleLeaf;
    beforeEach(async () => {
      leaf = await ctx.create(leafUpdate);
    });
    describe("internalUpdate", () => {
      it("should throw an error if the path is empty", async () => {
        await expect(
          leaf.internalUpdate({ ...leafUpdate, path: [] }),
        ).rejects.toThrowError(/empty path/);
      });
      it("should throw an error if the path has a subpath", async () => {
        await expect(
          async () =>
            await leaf.internalUpdate({ ...leafUpdate, path: ["test", "dog"] }),
        ).rejects.toThrowError(/subPath/);
      });
      it("should throw an error if the path does not have the correct key", async () => {
        await expect(
          leaf.internalUpdate({ ...leafUpdate, path: ["dog"] }),
        ).rejects.toThrowError(/key/);
      });
      it("should correctly internalUpdate the state", async () => {
        await leaf.internalUpdate({ ...leafUpdate, state: { x: 2 } });
        expect(leaf.state).toEqual({ x: 2 });
      });
      it("should call the handleUpdate", async () => {
        await leaf.internalUpdate({ ...leafUpdate, state: { x: 2 } });
        expect(leaf.updatef).toHaveBeenCalledTimes(2);
      });
    });
    describe("internalDelete", () => {
      it("should call the bound onDelete handler", async () => {
        await leaf.internalDelete(["test"]);
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
    beforeEach(async () => {
      composite = await ctx.create(compositeUpdate);
    });
    describe("setState", () => {
      it("should set the state of the composite's leaf if the path has one element", async () => {
        await composite.internalUpdate({ ...compositeUpdate, state: { x: 2 } });
      });
      it("should create a new leaf if the path has more than one element and the leaf does not exist", async () => {
        await composite.internalUpdate({
          ...leafUpdate,
          path: ["test", "dog"],
          state: { x: 2 },
        });
        expect(composite.children).toHaveLength(1);
        const c = composite.children[0];
        expect(c.key).toEqual("dog");
        expect(c.state).toEqual({ x: 2 });
      });
      it("should set the state of the composite's leaf if the path has more than one element and the leaf exists", async () => {
        // Create a child at the path.
        await composite.internalUpdate({
          ...leafUpdate,
          path: ["test", "dog"],
          state: { x: 2 },
        });
        // Update the state of the child.
        await composite.internalUpdate({
          ...leafUpdate,
          path: ["test", "dog"],
          state: { x: 3 },
        });
        expect(composite.children).toHaveLength(1);
        expect(composite.children[0].state).toEqual({ x: 3 });
      });
      it("should throw an error if the path is too deep and the child does not exist", async () => {
        await expect(
          composite.internalUpdate({
            ...compositeUpdate,
            path: ["test", "dog", "cat"],
          }),
        ).rejects.toThrowError();
      });
    });
    describe("internalDelete", () => {
      it("should remove a child from the list of children", async () => {
        await composite.internalUpdate({ ...compositeUpdate, path: ["test", "dog"] });
        await composite.internalDelete(["test", "dog"]);
        expect(composite.children).toHaveLength(0);
      });
      it("should call the deletion hook on the child of a composite", async () => {
        await composite.internalUpdate({ ...leafUpdate, path: ["test", "dog"] });
        const c = composite.children[0];
        await composite.internalDelete(["test", "dog"]);
        expect(c.deletef).toHaveBeenCalled();
      });
    });

    describe("context propagation", () => {
      it("should properly propagate an existing context change to its children", async () => {
        // Create two new leafs at separate paths.
        await composite.internalUpdate({ ...leafUpdate, path: ["test", "dog"] });
        await composite.internalUpdate({ ...leafUpdate, path: ["test", "cat"] });
        expect(composite.children).toHaveLength(2);
        composite.children.forEach((c) => {
          expect(c.updatef).toHaveBeenCalledTimes(1);
        });
        await composite.internalUpdate({ ...contextUpdate });
        expect(composite.updatef).toHaveBeenCalledTimes(2);
        composite.children.forEach((c) => {
          expect(c.updatef).toHaveBeenCalledTimes(2);
        });
      });

      it("should propagate a new context change to its children", async () => {
        const c = new ContextSetterComposite({ ...compositeUpdate });
        c.internalUpdate({ ...compositeUpdate });
        expect(c.ctx.get("key")).toEqual(1);
        await c.internalUpdate({ ...leafUpdate, path: ["test", "dog"] });
        expect(c.children).toHaveLength(1);
        c.children.forEach((c) => expect(c.ctx.get("key")).toEqual(1));
        await c.internalUpdate({ ...compositeUpdate, state: { x: 2 } });
        expect(c.children).toHaveLength(1);
        c.children.forEach((c) => {
          expect(c.updatef).toHaveBeenCalledTimes(2);
          expect(c.ctx.get("key")).toEqual(2);
        });
      });

      it("should correctly separate individual contexts", async () => {
        // Create two new context composites at "dog" and "cat". These will store
        // individual contexts.
        await composite.internalUpdate({
          ctx,
          variant: "state",
          type: "context",
          path: ["test", "dog"],
          state: { x: 2 },
          instrumentation: alamos.NOOP,
        });
        await composite.internalUpdate({
          ctx,
          variant: "state",
          type: "context",
          path: ["test", "cat"],
          state: { x: 3 },
          instrumentation: alamos.NOOP,
        });
        expect(composite.children).toHaveLength(2);
        // Assert that the two context setters have the correct, independent context values.
        const firstCtxSetter = composite.children[0] as ContextSetterComposite;
        const secondCtxSetter = composite.children[1] as ContextSetterComposite;
        expect(firstCtxSetter.ctx.get("key")).toEqual(2);
        expect(secondCtxSetter.ctx.get("key")).toEqual(3);
        // Create a new leaf for the "dog" context setter.
        await composite.internalUpdate({
          ctx,
          variant: "state",
          type: "leaf",
          path: ["test", "dog", "dogleaf"],
          state: { x: 3 },
          instrumentation: alamos.NOOP,
        });
        // Assert that the "dog" context setter has a child.
        expect(firstCtxSetter.children).toHaveLength(1);
        // Assert that the child has the correct context value.
        expect(firstCtxSetter.children[0].ctx.get("key")).toEqual(2);
        await composite.internalUpdate({
          ctx,
          variant: "state",
          type: "leaf",
          path: ["test", "cat", "catLeaf"],
          state: { x: 4 },
          instrumentation: alamos.NOOP,
        });
        // Assert that the "cat" context setter has a child.
        expect(secondCtxSetter.children).toHaveLength(1);
        // Assert that the child has the correct context value.
        expect(secondCtxSetter.children[0].ctx.get("key")).toEqual(3);
        await composite.internalUpdate({
          ctx,
          variant: "state",
          type: "context",
          path: ["test", "dog"],
          state: { x: 5 },
          instrumentation: alamos.NOOP,
        });
        expect(firstCtxSetter.children[0].ctx.get("key")).toEqual(5);
      });

      it.only("should correctly propagate a secondary context change as a result of a primary context change", async () => {
        // Create a primary context setter
        await composite.internalUpdate({
          ctx,
          variant: "state",
          type: "context",
          path: ["test", "primary"],
          state: { x: 2 },
          instrumentation: alamos.NOOP,
        });

        // Create a secondary context setter as a child of primary. This will set a new
        // context as a result of the primary context change.
        await composite.internalUpdate({
          ctx,
          variant: "state",
          type: "secondary",
          path: ["test", "primary", "secondary"],
          state: { x: 0 },
          instrumentation: alamos.NOOP,
        });

        // Create a leaf under the secondary context setter. This will set a new context as a
        // result of the secondary context change.
        await composite.internalUpdate({
          ctx,
          variant: "state",
          type: "leaf",
          path: ["test", "primary", "secondary", "leaf"],
          state: { x: 0 },
          instrumentation: alamos.NOOP,
        });

        const primarySetter = composite.children[0] as ContextSetterComposite;
        const secondarySetter = primarySetter.children[0] as SecondaryContextSetter;
        const leaf = secondarySetter.children[0] as ExampleLeaf;

        // Verify initial context values
        expect(primarySetter.ctx.get("key")).toEqual(2);
        expect(secondarySetter.ctx.get("key2")).toEqual(3);
        expect(leaf.ctx.get("key2")).toEqual(3);

        // Update the primary context setter
        await composite.internalUpdate({
          ctx,
          variant: "state",
          type: "state",
          path: ["test", "primary"],
          state: { x: 5 },
          instrumentation: alamos.NOOP,
        });

        // Verify that both the primary and secondary context values were updated
        expect(primarySetter.ctx.get("key")).toEqual(5);
        expect(secondarySetter.ctx.get("key2")).toEqual(6);
        expect(leaf.ctx.get("key2")).toEqual(6);
      });
    });
  });
});
