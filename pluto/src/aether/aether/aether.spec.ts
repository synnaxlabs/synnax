// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { scheduler } from "@synnaxlabs/x";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { aether } from "@/aether/aether";

const MockSender = {
  send: vi.fn(),
};

const createLeaf = (key: string, parent: aether.Component | null = null) =>
  new ExampleLeaf({
    path: [key],
    type: "leaf",
    sender: MockSender,
    instrumentation: alamos.Instrumentation.NOOP,
    parent,
  });

const createComposite = (key: string, parent: aether.Component | null = null) =>
  new ExampleComposite({
    path: [key],
    type: "composite",
    sender: MockSender,
    instrumentation: alamos.Instrumentation.NOOP,
    parent,
  });

const createContextSetter = (key: string, parent: aether.Component | null = null) =>
  new ContextSetterComposite({
    path: [key],
    type: "context",
    sender: MockSender,
    instrumentation: alamos.Instrumentation.NOOP,
    parent,
  });

const createSecondaryContextSetter = (
  key: string,
  parent: aether.Component | null = null,
) =>
  new SecondaryContextSetter({
    path: [key],
    type: "context",
    sender: MockSender,
    instrumentation: alamos.Instrumentation.NOOP,
    parent,
  });

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
  /** The values resolved for "key" and "key2" on the most recent afterUpdate. */
  seen: { key?: number; key2?: number } = {};

  afterUpdate(ctx: aether.Context): void {
    this.updatef(ctx);
    this.seen = {
      key: ctx.getOptional<number>("key") ?? undefined,
      key2: ctx.getOptional<number>("key2") ?? undefined,
    };
    this.internal.contextValue = this.seen.key ?? 0;
  }

  afterDelete(): void {
    this.deletef();
  }

  get testingChildCtxValues(): Map<string, any> {
    return this.childCtxValues;
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

  afterUpdate(ctx: aether.Context): void {
    this.updatef(ctx);
  }

  afterDelete(): void {
    this.deletef();
  }

  get testingChildCtxValues(): Map<string, any> {
    return this.childCtxValues;
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

  afterUpdate(ctx: aether.Context): void {
    this.updatef(ctx);
    ctx.set("key", this.state.x);
  }

  afterDelete(): void {
    this.deletef();
  }

  get testingChildCtxValues(): Map<string, any> {
    return this.childCtxValues;
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
  /** The value resolved for "key" on the most recent afterUpdate. */
  seen: { key?: number } = {};

  afterUpdate(ctx: aether.Context): void {
    this.updatef(ctx);
    const v = ctx.getOptional<number>("key");
    this.seen = { key: v ?? undefined };
    if (v != null) ctx.set("key2", v + 1);
  }

  get testingChildCtxValues(): Map<string, any> {
    return this.childCtxValues;
  }
}

/** Reads "key" only when `state.x > 0`, so its subscription appears and disappears with
 * state. */
class ConditionalReader extends aether.Leaf<typeof exampleProps> {
  updatef = vi.fn();
  schema = exampleProps;
  seen: number | undefined = undefined;

  afterUpdate(ctx: aether.Context): void {
    this.updatef(ctx);
    this.seen =
      this.state.x > 0 ? (ctx.getOptional<number>("key") ?? undefined) : undefined;
  }
}

const createConditionalReader = (key: string, parent: aether.Component | null = null) =>
  new ConditionalReader({
    path: [key],
    type: "conditional",
    sender: MockSender,
    instrumentation: alamos.Instrumentation.NOOP,
    parent,
  });

/** Republishes "key" as `state.x * 10`, shadowing an ancestor that also publishes it. */
class ShadowingSetter extends aether.Composite<typeof exampleProps, {}, ExampleLeaf> {
  updatef = vi.fn();
  schema = exampleProps;

  afterUpdate(ctx: aether.Context): void {
    this.updatef(ctx);
    ctx.set("key", this.state.x * 10);
  }
}

const createShadowingSetter = (key: string, parent: aether.Component | null = null) =>
  new ShadowingSetter({
    path: [key],
    type: "shadow",
    sender: MockSender,
    instrumentation: alamos.Instrumentation.NOOP,
    parent,
  });

/** Re-publishes a constant value with `trigger=true` on every update — the canvas
 * pattern. Subscribers must re-run even though the value is reference-equal. */
class InstanceSetter extends aether.Composite<typeof exampleProps, {}, ExampleLeaf> {
  updatef = vi.fn();
  schema = exampleProps;

  afterUpdate(ctx: aether.Context): void {
    this.updatef(ctx);
    ctx.set("key", 7);
  }
}

const createInstanceSetter = (key: string, parent: aether.Component | null = null) =>
  new InstanceSetter({
    path: [key],
    type: "instance",
    sender: MockSender,
    instrumentation: alamos.Instrumentation.NOOP,
    parent,
  });

/** Publishes "late" only once `state.x > 0`, i.e. potentially for the first time after
 * mount — exercises the late-shadowing guard. */
class LateSetter extends aether.Composite<typeof exampleProps, {}, ExampleLeaf> {
  updatef = vi.fn();
  schema = exampleProps;

  afterUpdate(ctx: aether.Context): void {
    this.updatef(ctx);
    if (this.state.x > 0) ctx.set("late", this.state.x);
  }
}

const createLateSetter = (key: string, parent: aether.Component | null = null) =>
  new LateSetter({
    path: [key],
    type: "late",
    sender: MockSender,
    instrumentation: alamos.Instrumentation.NOOP,
    parent,
  });

const shouldNotCallCreate = () => {
  throw new Error("should not call create");
};

const invokeMethodsSchema = {
  increment: z.function({ input: z.tuple([z.number()]), output: z.number() }),
  greet: z.function({
    input: z.tuple([z.object({ name: z.string() })]),
    output: z.string(),
  }),
  noArgs: z.function(),
  asyncMethod: z.function({
    input: z.tuple([z.number()]),
    output: z.promise(z.number()),
  }),
  throwError: z.function(),
} satisfies aether.MethodsSchema;

class InvokeLeaf
  extends aether.Leaf<typeof exampleProps, {}, typeof invokeMethodsSchema>
  implements aether.HandlersFromSchema<typeof invokeMethodsSchema>
{
  schema = exampleProps;
  methods = invokeMethodsSchema;

  // Track calls for testing
  incrementSpy = vi.fn((n: number) => n + 1);
  greetSpy = vi.fn((args: { name: string }) => `Hello, ${args.name}!`);
  noArgsSpy = vi.fn(() => {});
  asyncMethodSpy = vi.fn(async (n: number) => {
    await new Promise((resolve) => setTimeout(resolve, 10));
    return n * 2;
  });
  throwErrorSpy = vi.fn(() => {
    throw new Error("Test error");
  });

  // Methods matching the schema
  increment(n: number): number {
    return this.incrementSpy(n);
  }

  greet(args: { name: string }): string {
    return this.greetSpy(args);
  }

  noArgs(): void {
    this.noArgsSpy();
  }

  asyncMethod(n: number): Promise<number> {
    return this.asyncMethodSpy(n);
  }

  throwError(): void {
    this.throwErrorSpy();
  }

  afterUpdate(): void {}
  afterDelete(): void {}
}

const createInvokeLeaf = (key: string, parent: aether.Component | null = null) =>
  new InvokeLeaf({
    path: [key],
    type: "invoke-leaf",
    sender: MockSender,
    instrumentation: alamos.Instrumentation.NOOP,
    parent,
  });

class InvokeComposite extends aether.Composite<
  typeof exampleProps,
  {},
  InvokeLeaf,
  aether.EmptyMethodsSchema
> {
  schema = exampleProps;
  methods = undefined;

  afterUpdate(): void {}
  afterDelete(): void {}
}

const createInvokeComposite = (key: string, parent: aether.Component | null = null) =>
  new InvokeComposite({
    path: [key],
    type: "invoke-composite",
    sender: MockSender,
    instrumentation: alamos.Instrumentation.NOOP,
    parent,
  });

describe("Aether Worker", () => {
  describe("AetherLeaf", () => {
    let leaf: ExampleLeaf;
    beforeEach(async () => {
      leaf = createLeaf("test");
    });

    describe("internalUpdate", () => {
      it("should throw an error if the path is empty", async () => {
        expect(() => {
          leaf._updateState({
            path: [],
            state: {},
            type: "example",
            create: (parent) => createLeaf("test", parent),
          });
        }).toThrow(/empty path/);
        expect(leaf.updatef).toHaveBeenCalledTimes(0);
      });

      it("should throw an error if the path has a subpath", async () => {
        expect(() => {
          leaf._updateState({
            path: ["test", "dog"],
            state: {},
            type: "example",
            create: (parent) => createLeaf("dog", parent),
          });
        }).toThrow(/subPath/);
        expect(leaf.updatef).toHaveBeenCalledTimes(0);
      });

      it("should throw an error if the path does not have the correct key", async () => {
        expect(() => {
          leaf._updateState({
            path: ["dog"],
            state: {},
            type: "example",
            create: (parent) => createLeaf("dog", parent),
          });
        }).toThrow(/key/);
        expect(leaf.updatef).toHaveBeenCalledTimes(0);
      });

      it("should correctly internalUpdate the state", async () => {
        leaf._updateState({
          path: ["test"],
          state: { x: 2 },
          type: "example",
          create: (parent) => createLeaf("test", parent),
        });
        expect(leaf.state).toEqual({ x: 2 });
      });

      it("should call the handleUpdate", async () => {
        leaf._updateState({
          path: ["test"],
          state: { x: 2 },
          type: "example",
          create: (parent) => createLeaf("test", parent),
        });
        expect(leaf.updatef).toHaveBeenCalledTimes(1);
      });
    });

    describe("internalDelete", () => {
      it("should call the bound onDelete handler", async () => {
        leaf._delete(["test"]);
        expect(leaf.deletef).toHaveBeenCalledTimes(1);
      });
    });

    describe("setState", () => {
      it("should communicate the state call to the main thread Sender", async () => {
        leaf._updateState({
          path: ["test"],
          state: { x: 2 },
          type: "example",
          create: (parent) => createLeaf("test", parent),
        });
        leaf.setState((p) => ({ ...p }));
        expect(MockSender.send).toHaveBeenCalledTimes(1);
        expect(MockSender.send).toHaveBeenCalledWith({
          variant: "update",
          path: ["test"],
          state: { x: 2 },
        });
      });
    });
  });

  describe("AetherComposite", () => {
    let composite: ExampleComposite;
    beforeEach(async () => {
      composite = createComposite("test");
    });

    describe("setState", () => {
      it("should set the state of the composite itself if the path has one element", async () => {
        composite._updateState({
          path: ["test"],
          state: { x: 2 },
          type: "example",
          create: shouldNotCallCreate,
        });
        expect(composite.state).toEqual({ x: 2 });
        expect(composite.updatef).toHaveBeenCalledTimes(1);
      });

      it("should create a new leaf if the path has more than one element and the leaf does not exist", async () => {
        composite._updateState({
          path: ["test", "dog"],
          state: { x: 2 },
          type: "example",
          create: () => createLeaf("dog"),
        });
        expect(composite.children).toHaveLength(1);
        const c = composite.children[0];
        expect(c.key).toEqual("dog");
        expect(c.state).toEqual({ x: 2 });
        expect(c.updatef).toHaveBeenCalledTimes(1);
      });

      it("should set the state of the composite's leaf if the path has more than one element and the leaf exists", async () => {
        composite._updateState({
          path: ["test", "dog"],
          state: { x: 2 },
          type: "example",
          create: () => createLeaf("dog"),
        });
        composite._updateState({
          path: ["test", "dog"],
          state: { x: 3 },
          type: "example",
          create: shouldNotCallCreate,
        });
        expect(composite.children).toHaveLength(1);
        expect(composite.children[0].state).toEqual({ x: 3 });
      });

      it("should throw an error if the path is too deep and the child does not exist", async () => {
        expect(() => {
          composite._updateState({
            path: ["test", "dog", "cat"],
            state: { x: 2 },
            type: "example",
            create: shouldNotCallCreate,
          });
        }).toThrow(/attempting to create a new child/);
      });
    });

    describe("internalDelete", () => {
      it("should remove a child from the list of children", async () => {
        composite._updateState({
          path: ["test", "dog"],
          state: { x: 2 },
          type: "example",
          create: () => createLeaf("dog"),
        });
        expect(composite.children).toHaveLength(1);
        composite._delete(["test", "dog"]);
        expect(composite.children).toHaveLength(0);
      });

      it("should call the deletion hook on the child of a composite", async () => {
        composite._updateState({
          path: ["test", "dog"],
          state: { x: 2 },
          type: "example",
          create: () => createLeaf("dog"),
        });
        const c = composite.children[0];
        composite._delete(["test", "dog"]);
        expect(c.deletef).toHaveBeenCalled();
      });
    });
  });

  describe("context propagation", () => {
    it("should correctly set a context value", async () => {
      const v = createContextSetter("test");
      v._updateState({
        path: ["test"],
        state: { x: 2 },
        type: "example",
        create: shouldNotCallCreate,
      });
      expect(v.testingChildCtxValues.get("key")).toEqual(2);
    });

    it("should correctly pass an initial context value to a leaf child", async () => {
      const v = createContextSetter("test");
      v._updateState({
        path: ["test"],
        state: { x: 2 },
        type: "example",
        create: shouldNotCallCreate,
      });
      v._updateState({
        path: ["test", "dog"],
        state: { x: 3 },
        type: "example",
        create: (c) => createLeaf("dog", c),
      });
      const c = v.children[0] as ExampleLeaf;
      expect(c.seen.key).toEqual(2);
      expect(c.testingChildCtxValues.size).toEqual(0);
    });

    it("should correctly update the context value in a child leaf", async () => {
      const v = createContextSetter("test");
      v._updateState({
        path: ["test"],
        state: { x: 2 },
        type: "example",
        create: shouldNotCallCreate,
      });
      v._updateState({
        path: ["test", "dog"],
        state: { x: 3 },
        type: "example",
        create: (c) => createLeaf("dog", c),
      });
      const c = v.children[0] as ExampleLeaf;
      v._updateState({
        path: ["test"],
        state: { x: 4 },
        type: "example",
        create: shouldNotCallCreate,
      });
      expect(v.testingChildCtxValues.get("key")).toEqual(4);
      expect(c.seen.key).toEqual(4);
      expect(c.testingChildCtxValues.size).toEqual(0);
    });

    it("should correctly separate individual contexts", async () => {
      const v = createComposite("test");
      v._updateState({
        path: ["test"],
        state: { x: 2 },
        type: "example",
        create: shouldNotCallCreate,
      });
      v._updateState({
        path: ["test", "dog"],
        state: { x: 3 },
        type: "example",
        create: (c) => createContextSetter("dog", c),
      });
      v._updateState({
        path: ["test", "cat"],
        state: { x: 4 },
        type: "example",
        create: (c) => createContextSetter("cat", c),
      });
      expect(v.children).toHaveLength(2);
      const c1 = v.children[0];
      const c2 = v.children[1];
      expect(c1.testingChildCtxValues.size).toEqual(1);
      expect(c2.testingChildCtxValues.size).toEqual(1);
      expect(c1.testingChildCtxValues.get("key")).toEqual(3);
      expect(c2.testingChildCtxValues.get("key")).toEqual(4);
    });

    it("should correctly initialize contexts with a nested leaf", async () => {
      const v = createContextSetter("first");
      v._updateState({
        path: ["first"],
        state: { x: 2 },
        type: "example",
        create: shouldNotCallCreate,
      });
      expect(v.testingChildCtxValues.size).toEqual(1);
      v._updateState({
        path: ["first", "second"],
        state: { x: 3 },
        type: "example",
        create: (c) => createSecondaryContextSetter("second", c),
      });
      const c1 = v.children[0] as SecondaryContextSetter;
      expect(c1.seen.key).toEqual(2);
      expect(c1.testingChildCtxValues.get("key2")).toEqual(3);
      v._updateState({
        path: ["first", "second", "third"],
        state: { x: 4 },
        type: "example",
        create: (c) => createLeaf("third", c),
      });
      const c2 = (v.children[0] as SecondaryContextSetter).children[0];
      expect(c2.testingChildCtxValues.size).toEqual(0);
      expect(c2.seen.key).toEqual(2);
      expect(c2.seen.key2).toEqual(3);
    });

    it("should correctly update contexts with a nested leaf", async () => {
      const v = createContextSetter("first");
      v._updateState({
        path: ["first"],
        state: { x: 2 },
        type: "example",
        create: shouldNotCallCreate,
      });
      v._updateState({
        path: ["first", "second"],
        state: { x: 3 },
        type: "example",
        create: (c) => createSecondaryContextSetter("second", c),
      });
      const c1 = v.children[0] as SecondaryContextSetter;
      v._updateState({
        path: ["first", "second", "third"],
        state: { x: 4 },
        type: "example",
        create: (c) => createLeaf("third", c),
      });
      const c2 = (v.children[0] as SecondaryContextSetter).children[0];
      v._updateState({
        path: ["first"],
        state: { x: 5 },
        type: "example",
        create: shouldNotCallCreate,
      });
      expect(c1.seen.key).toEqual(5);
      expect(c1.testingChildCtxValues.get("key2")).toEqual(6);
      expect(c2.seen.key).toEqual(5);
      expect(c2.seen.key2).toEqual(6);
    });

    it("re-runs only the subscribers of a changed key, not the whole subtree", () => {
      const root = createContextSetter("root");
      root._updateState({
        path: ["root"],
        state: { x: 1 },
        type: "example",
        create: shouldNotCallCreate,
      });
      // reader subscribes to "key"; ignorer never reads context.
      let reader!: ExampleLeaf;
      let ignorer!: ExampleComposite;
      root._updateState({
        path: ["root", "reader"],
        state: { x: 0 },
        type: "example",
        create: (c) => (reader = createLeaf("reader", c)),
      });
      root._updateState({
        path: ["root", "ignorer"],
        state: { x: 0 },
        type: "example",
        create: (c) => (ignorer = createComposite("ignorer", c)),
      });
      reader.updatef.mockClear();
      ignorer.updatef.mockClear();
      root._updateState({
        path: ["root"],
        state: { x: 2 },
        type: "example",
        create: shouldNotCallCreate,
      });
      expect(reader.updatef).toHaveBeenCalledTimes(1);
      expect(reader.seen.key).toEqual(2);
      expect(ignorer.updatef).not.toHaveBeenCalled();
    });

    it("subscribes and unsubscribes as a conditional read appears and disappears", () => {
      const root = createContextSetter("root");
      root._updateState({
        path: ["root"],
        state: { x: 1 },
        type: "example",
        create: shouldNotCallCreate,
      });
      // ConditionalReader reads "key" only when state.x > 0.
      let cond!: ConditionalReader;
      root._updateState({
        path: ["root", "cond"],
        state: { x: 1 },
        type: "example",
        create: (c) => (cond = createConditionalReader("cond", c)),
      });
      expect(cond.seen).toEqual(1);

      // Stop reading "key" (x -> 0). This re-runs cond and unsubscribes it.
      root._updateState({
        path: ["root", "cond"],
        state: { x: 0 },
        type: "example",
        create: shouldNotCallCreate,
      });
      cond.updatef.mockClear();
      root._updateState({
        path: ["root"],
        state: { x: 5 },
        type: "example",
        create: shouldNotCallCreate,
      });
      expect(cond.updatef).not.toHaveBeenCalled();

      // Resume reading "key" (x -> 1); now changes to "key" re-run it again.
      root._updateState({
        path: ["root", "cond"],
        state: { x: 1 },
        type: "example",
        create: shouldNotCallCreate,
      });
      cond.updatef.mockClear();
      root._updateState({
        path: ["root"],
        state: { x: 9 },
        type: "example",
        create: shouldNotCallCreate,
      });
      expect(cond.updatef).toHaveBeenCalledTimes(1);
      expect(cond.seen).toEqual(9);
    });

    it("binds a consumer to the nearest provider when a key is shadowed", () => {
      const root = createContextSetter("root");
      root._updateState({
        path: ["root"],
        state: { x: 1 },
        type: "example",
        create: shouldNotCallCreate,
      });
      // mid republishes "key" (x * 10), shadowing root for its descendants.
      let mid!: ShadowingSetter;
      root._updateState({
        path: ["root", "mid"],
        state: { x: 2 },
        type: "example",
        create: (c) => (mid = createShadowingSetter("mid", c)),
      });
      root._updateState({
        path: ["root", "mid", "leaf"],
        state: { x: 0 },
        type: "example",
        create: (c) => createLeaf("leaf", c),
      });
      const leaf = mid.children[0];
      expect(leaf.seen.key).toEqual(20);

      // Changing the far provider (root) does NOT re-run the shadowed leaf.
      leaf.updatef.mockClear();
      root._updateState({
        path: ["root"],
        state: { x: 7 },
        type: "example",
        create: shouldNotCallCreate,
      });
      expect(leaf.updatef).not.toHaveBeenCalled();
      expect(leaf.seen.key).toEqual(20);

      // Changing the near provider (mid) does re-run it.
      leaf.updatef.mockClear();
      root._updateState({
        path: ["root", "mid"],
        state: { x: 3 },
        type: "example",
        create: shouldNotCallCreate,
      });
      expect(leaf.updatef).toHaveBeenCalledTimes(1);
      expect(leaf.seen.key).toEqual(30);
    });

    it("cascades to subscribers when a provider re-sets the same value reference", () => {
      const root = createInstanceSetter("root");
      root._updateState({
        path: ["root"],
        state: { x: 1 },
        type: "example",
        create: shouldNotCallCreate,
      });
      root._updateState({
        path: ["root", "leaf"],
        state: { x: 0 },
        type: "example",
        create: (c) => createLeaf("leaf", c),
      });
      const leaf = root.children[0];
      leaf.updatef.mockClear();
      // A self-update re-runs root.afterUpdate, which re-sets the SAME instance with
      // trigger=true (the canvas pattern). Subscribers must still re-run.
      root._updateState({
        path: ["root"],
        state: { x: 2 },
        type: "example",
        create: shouldNotCallCreate,
      });
      expect(leaf.updatef).toHaveBeenCalledTimes(1);
    });

    it("unsubscribes a deleted consumer so it is not re-run", () => {
      const root = createContextSetter("root");
      root._updateState({
        path: ["root"],
        state: { x: 1 },
        type: "example",
        create: shouldNotCallCreate,
      });
      root._updateState({
        path: ["root", "leaf"],
        state: { x: 0 },
        type: "example",
        create: (c) => createLeaf("leaf", c),
      });
      const leaf = root.children[0] as ExampleLeaf;
      root._delete(["root", "leaf"]);
      leaf.updatef.mockClear();
      root._updateState({
        path: ["root"],
        state: { x: 2 },
        type: "example",
        create: shouldNotCallCreate,
      });
      expect(leaf.updatef).not.toHaveBeenCalled();
    });

    it("warns when a provider publishes a key for the first time after mount", () => {
      const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
      try {
        const root = createLateSetter("root");
        root._updateState({
          path: ["root"],
          state: { x: 0 },
          type: "example",
          create: shouldNotCallCreate,
        });
        // Give it a child so the late-shadow guard's hasChildren check is satisfied.
        root._updateState({
          path: ["root", "leaf"],
          state: { x: 0 },
          type: "example",
          create: (c) => createLeaf("leaf", c),
        });
        expect(warn).not.toHaveBeenCalled();
        // x -> 1 makes root publish "late" for the first time.
        root._updateState({
          path: ["root"],
          state: { x: 1 },
          type: "example",
          create: shouldNotCallCreate,
        });
        expect(warn).toHaveBeenCalledTimes(1);
        expect(warn.mock.calls[0][0]).toContain("late context shadowing");
      } finally {
        warn.mockRestore();
      }
    });
  });

  describe("invoke", () => {
    let leaf: InvokeLeaf;
    beforeEach(() => {
      MockSender.send.mockClear();
      leaf = createInvokeLeaf("invoke-test");
      leaf._updateState({
        path: ["invoke-test"],
        state: { x: 1 },
        type: "invoke-leaf",
        create: shouldNotCallCreate,
      });
    });

    describe("_invokeMethod", () => {
      it("should invoke the handler with the provided args and send response", async () => {
        leaf._invokeMethod({
          key: "req-1",
          path: [],
          args: [5],
          method: "increment",
        });
        await scheduler.flushTaskQueue();

        expect(leaf.incrementSpy).toHaveBeenCalledWith(5);
        expect(MockSender.send).toHaveBeenCalledWith({
          variant: "invoke_response",
          key: "req-1",
          result: 6,
        });
      });

      it("should handle methods with object args", async () => {
        leaf._invokeMethod({
          key: "req-2",
          path: [],
          args: [{ name: "World" }],
          method: "greet",
        });
        await scheduler.flushTaskQueue();

        expect(leaf.greetSpy).toHaveBeenCalledWith({ name: "World" });
        expect(MockSender.send).toHaveBeenCalledWith({
          variant: "invoke_response",
          key: "req-2",
          result: "Hello, World!",
        });
      });

      it("should handle methods with no args (fire-and-forget)", async () => {
        leaf._invokeMethod({
          path: [],
          args: [],
          method: "noArgs",
        });
        await scheduler.flushTaskQueue();

        expect(leaf.noArgsSpy).toHaveBeenCalled();
        expect(MockSender.send).not.toHaveBeenCalled();
      });

      it("should handle async methods", async () => {
        leaf._invokeMethod({
          key: "req-4",
          path: [],
          args: [10],
          method: "asyncMethod",
        });

        await new Promise((resolve) => setTimeout(resolve, 20));

        expect(leaf.asyncMethodSpy).toHaveBeenCalledWith(10);
        expect(MockSender.send).toHaveBeenCalledWith({
          variant: "invoke_response",
          key: "req-4",
          result: 20,
        });
      });

      it("should send error response when handler throws (key defined)", async () => {
        leaf._invokeMethod({
          key: "req-5",
          path: [],
          args: [],
          method: "throwError",
        });

        await scheduler.flushTaskQueue();

        expect(leaf.throwErrorSpy).toHaveBeenCalled();
        expect(MockSender.send).toHaveBeenCalledWith({
          variant: "invoke_response",
          key: "req-5",
          result: undefined,
          error: expect.objectContaining({
            name: "Error",
            data: "Failed to execute throwError(req-5) with args [] on invoke-leaf(invoke-test): Test error",
          }),
        });
      });

      it("should log error but not send response when handler throws (fire-and-forget)", async () => {
        const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
        leaf._invokeMethod({
          path: [],
          args: [],
          method: "throwError",
        });
        await scheduler.flushTaskQueue();

        expect(leaf.throwErrorSpy).toHaveBeenCalled();
        expect(MockSender.send).not.toHaveBeenCalled();
        expect(consoleSpy).toHaveBeenCalled();
        consoleSpy.mockRestore();
      });

      it("should send error response for unknown method when key defined", () => {
        leaf._invokeMethod({
          key: "req-6",
          path: [],
          args: [],
          method: "unknownMethod",
        });

        expect(MockSender.send).toHaveBeenCalledWith({
          variant: "invoke_response",
          key: "req-6",
          result: undefined,
          error: expect.objectContaining({
            data: expect.stringContaining("unknownMethod"),
          }),
        });
      });

      it("should not invoke method if component is deleted", () => {
        leaf._delete(["invoke-test"]);
        MockSender.send.mockClear();

        leaf._invokeMethod({
          key: "req-7",
          path: [],
          args: [5],
          method: "increment",
        });

        expect(leaf.incrementSpy).not.toHaveBeenCalled();
        expect(MockSender.send).not.toHaveBeenCalled();
      });
    });

    describe("Composite invoke propagation", () => {
      let composite: InvokeComposite;
      let childLeaf: InvokeLeaf;

      beforeEach(() => {
        MockSender.send.mockClear();
        composite = createInvokeComposite("parent");
        composite._updateState({
          path: ["parent"],
          state: { x: 1 },
          type: "invoke-composite",
          create: shouldNotCallCreate,
        });
        composite._updateState({
          path: ["parent", "child"],
          state: { x: 2 },
          type: "invoke-leaf",
          create: () => createInvokeLeaf("child"),
        });
        childLeaf = composite.children[0];
        MockSender.send.mockClear();
      });

      it("should find child at path using findChildAtPath", () => {
        const found = composite.findChildAtPath(["child"]);
        expect(found).toBe(childLeaf);
      });

      it("should return null for non-existent path", () => {
        const found = composite.findChildAtPath(["non-existent"]);
        expect(found).toBeNull();
      });

      it("should return null for empty path", () => {
        const found = composite.findChildAtPath([]);
        expect(found).toBeNull();
      });
    });
  });
});

describe("message", () => {
  describe("NOOP_MAIN_COMMS", () => {
    it("should provide no-op send and handle", () => {
      expect(() =>
        aether.NOOP_MAIN_COMMS.send({ variant: "update" } as any),
      ).not.toThrow();
      expect(() => aether.NOOP_MAIN_COMMS.handle(() => {})).not.toThrow();
    });
  });

  describe("wrapWorker", () => {
    it("should forward send to worker.postMessage with transfer", () => {
      const postMessage = vi.fn();
      const worker = { postMessage, onmessage: null } as unknown as Worker;
      const comms = aether.wrapWorker(worker);
      const msg: aether.MainMessage = {
        variant: "update",
        path: ["a"],
        type: "t",
        state: { x: 1 },
      };
      const transfer: Transferable[] = [];
      comms.send(msg, transfer);
      expect(postMessage).toHaveBeenCalledWith(msg, transfer);
    });

    it("should default transfer to empty array when omitted", () => {
      const postMessage = vi.fn();
      const worker = { postMessage, onmessage: null } as unknown as Worker;
      const comms = aether.wrapWorker(worker);
      const msg: aether.MainMessage = {
        variant: "delete",
        path: ["a"],
      };
      comms.send(msg);
      expect(postMessage).toHaveBeenCalledWith(msg, []);
    });

    it("should route worker.onmessage events to the registered handler", () => {
      const worker = { postMessage: vi.fn(), onmessage: null } as unknown as Worker;
      const comms = aether.wrapWorker(worker);
      const handler = vi.fn();
      comms.handle(handler);
      const msg: aether.WorkerMessage = {
        variant: "update",
        path: ["k"],
        state: { x: 1 },
      };
      (worker as any).onmessage({ data: msg });
      expect(handler).toHaveBeenCalledWith(msg);
    });
  });

  describe("wrapWorkerScope", () => {
    it("should forward send to global postMessage with transfer", () => {
      const postMessage = vi.fn();
      vi.stubGlobal("postMessage", postMessage);
      try {
        const comms = aether.wrapWorkerScope();
        const msg: aether.WorkerMessage = {
          variant: "update",
          path: ["k"],
          state: { x: 1 },
        };
        const transfer: Transferable[] = [];
        comms.send(msg, transfer);
        expect(postMessage).toHaveBeenCalledWith(msg, { transfer });
      } finally {
        vi.unstubAllGlobals();
      }
    });

    it("should route global onmessage events to the registered handler", () => {
      vi.stubGlobal("onmessage", null);
      try {
        const comms = aether.wrapWorkerScope();
        const handler = vi.fn();
        comms.handle(handler);
        const msg: aether.MainMessage = {
          variant: "update",
          path: ["a"],
          type: "t",
          state: { x: 1 },
        };
        (globalThis as any).onmessage({ data: msg });
        expect(handler).toHaveBeenCalledWith(msg);
      } finally {
        vi.unstubAllGlobals();
      }
    });
  });

  describe("createMockPair", () => {
    it("should deliver main-side sends to the worker-side handler", () => {
      const [workerSide, mainSide] = aether.createMockPair();
      const workerHandler = vi.fn();
      workerSide.handle(workerHandler);
      const msg: aether.MainMessage = {
        variant: "update",
        path: ["a"],
        type: "t",
        state: { x: 1 },
      };
      mainSide.send(msg);
      expect(workerHandler).toHaveBeenCalledWith(msg);
    });

    it("should deliver worker-side sends to the main-side handler", () => {
      const [workerSide, mainSide] = aether.createMockPair();
      const mainHandler = vi.fn();
      mainSide.handle(mainHandler);
      const msg: aether.WorkerMessage = {
        variant: "update",
        path: ["k"],
        state: { x: 1 },
      };
      workerSide.send(msg);
      expect(mainHandler).toHaveBeenCalledWith(msg);
    });

    it("should drop sends made before a handler is registered", () => {
      const [workerSide, mainSide] = aether.createMockPair();
      expect(() =>
        mainSide.send({
          variant: "update",
          path: ["a"],
          type: "t",
          state: { x: 1 },
        }),
      ).not.toThrow();
      const workerHandler = vi.fn();
      workerSide.handle(workerHandler);
      expect(workerHandler).not.toHaveBeenCalled();
    });

    it("should replace the previously registered handler on re-handle", () => {
      const [workerSide, mainSide] = aether.createMockPair();
      const first = vi.fn();
      const second = vi.fn();
      workerSide.handle(first);
      workerSide.handle(second);
      mainSide.send({
        variant: "delete",
        path: ["a"],
      });
      expect(first).not.toHaveBeenCalled();
      expect(second).toHaveBeenCalledTimes(1);
    });
  });
});
