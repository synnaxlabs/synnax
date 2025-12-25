// Copyright 2025 Synnax Labs, Inc.
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

const createLeaf = (key: string, parentCtxValues: aether.ContextMap | null = null) =>
  new ExampleLeaf({
    key,
    type: "leaf",
    sender: MockSender,
    instrumentation: alamos.Instrumentation.NOOP,
    parentCtxValues,
  });

const createComposite = (
  key: string,
  parentCtxValues: Map<string, any> | null = null,
) =>
  new ExampleComposite({
    key,
    type: "composite",
    sender: MockSender,
    instrumentation: alamos.Instrumentation.NOOP,
    parentCtxValues,
  });

const createContextSetter = (
  key: string,
  parentCtxValues: aether.ContextMap | null = null,
) =>
  new ContextSetterComposite({
    key,
    type: "context",
    sender: MockSender,
    instrumentation: alamos.Instrumentation.NOOP,
    parentCtxValues,
  });

const createSecondaryContextSetter = (
  key: string,
  parentCtxValues: aether.ContextMap | null = null,
) =>
  new SecondaryContextSetter({
    key,
    type: "context",
    sender: MockSender,
    instrumentation: alamos.Instrumentation.NOOP,
    parentCtxValues,
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

  afterUpdate(ctx: aether.Context): void {
    this.updatef(ctx);
    this.internal.contextValue = ctx.getOptional("key") ?? 0;
  }

  afterDelete(): void {
    this.deletef();
  }

  get testingParentCtxValues(): Map<string, any> {
    return this.parentCtxValues;
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

  get testingParentCtxValues(): Map<string, any> {
    return this.parentCtxValues;
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

  get testingParentCtxValues(): Map<string, any> {
    return this.parentCtxValues;
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

  afterUpdate(ctx: aether.Context): void {
    this.updatef(ctx);
    const v = ctx.getOptional<number>("key");
    if (v != null) ctx.set("key2", v + 1);
  }

  get testingParentCtxValues(): Map<string, any> {
    return this.parentCtxValues;
  }

  get testingChildCtxValues(): Map<string, any> {
    return this.childCtxValues;
  }
}

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

const createInvokeLeaf = (
  key: string,
  parentCtxValues: aether.ContextMap | null = null,
) =>
  new InvokeLeaf({
    key,
    type: "invoke-leaf",
    sender: MockSender,
    instrumentation: alamos.Instrumentation.NOOP,
    parentCtxValues,
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

const createInvokeComposite = (
  key: string,
  parentCtxValues: aether.ContextMap | null = null,
) =>
  new InvokeComposite({
    key,
    type: "invoke-composite",
    sender: MockSender,
    instrumentation: alamos.Instrumentation.NOOP,
    parentCtxValues,
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
            create: (parentCtxValues) => createLeaf("test", parentCtxValues),
          });
        }).toThrowError(/empty path/);
        expect(leaf.updatef).toHaveBeenCalledTimes(0);
      });

      it("should throw an error if the path has a subpath", async () => {
        expect(() => {
          leaf._updateState({
            path: ["test", "dog"],
            state: {},
            type: "example",
            create: (parentCtxValues) => createLeaf("dog", parentCtxValues),
          });
        }).toThrowError(/subPath/);
        expect(leaf.updatef).toHaveBeenCalledTimes(0);
      });

      it("should throw an error if the path does not have the correct key", async () => {
        expect(() => {
          leaf._updateState({
            path: ["dog"],
            state: {},
            type: "example",
            create: (parentCtxValues) => createLeaf("dog", parentCtxValues),
          });
        }).toThrowError(/key/);
        expect(leaf.updatef).toHaveBeenCalledTimes(0);
      });

      it("should correctly internalUpdate the state", async () => {
        leaf._updateState({
          path: ["test"],
          state: { x: 2 },
          type: "example",
          create: (parentCtxValues) => createLeaf("test", parentCtxValues),
        });
        expect(leaf.state).toEqual({ x: 2 });
      });

      it("should call the handleUpdate", async () => {
        leaf._updateState({
          path: ["test"],
          state: { x: 2 },
          type: "example",
          create: (parentCtxValues) => createLeaf("test", parentCtxValues),
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
          create: (parentCtxValues) => createLeaf("test", parentCtxValues),
        });
        leaf.setState((p) => ({ ...p }));
        expect(MockSender.send).toHaveBeenCalledTimes(1);
        expect(MockSender.send).toHaveBeenCalledWith({
          variant: "update",
          key: "test",
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
        }).toThrowError(/attempting to create a new child/);
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
      expect(v.testingParentCtxValues.size).toEqual(0);
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
      const c = v.children[0];
      expect(c.testingParentCtxValues.get("key")).toEqual(2);
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
      const c = v.children[0];
      v._updateState({
        path: ["test"],
        state: { x: 4 },
        type: "example",
        create: shouldNotCallCreate,
      });
      expect(v.testingChildCtxValues.get("key")).toEqual(4);
      expect(v.testingParentCtxValues.size).toEqual(0);
      expect(c.testingParentCtxValues.get("key")).toEqual(4);
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
      const c1 = v.children[0];
      expect(c1.testingParentCtxValues.size).toEqual(1);
      expect(c1.testingChildCtxValues.size).toEqual(1);
      v._updateState({
        path: ["first", "second", "third"],
        state: { x: 4 },
        type: "example",
        create: (c) => createLeaf("third", c),
      });
      const c2 = (v.children[0] as SecondaryContextSetter).children[0];
      expect(c2.testingParentCtxValues.size).toEqual(2);
      expect(c2.testingChildCtxValues.size).toEqual(0);
      expect(c2.testingParentCtxValues.get("key")).toEqual(2);
      expect(c2.testingParentCtxValues.get("key2")).toEqual(3);
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
        create: () => createSecondaryContextSetter("second"),
      });
      const c1 = v.children[0];
      v._updateState({
        path: ["first", "second", "third"],
        state: { x: 4 },
        type: "example",
        create: () => createLeaf("third"),
      });
      const c2 = (v.children[0] as SecondaryContextSetter).children[0];
      v._updateState({
        path: ["first"],
        state: { x: 5 },
        type: "example",
        create: shouldNotCallCreate,
      });
      expect(c1.testingParentCtxValues.size).toEqual(1);
      expect(c1.testingChildCtxValues.size).toEqual(1);
      expect(c2.testingParentCtxValues.size).toEqual(2);
      expect(c2.testingChildCtxValues.size).toEqual(0);
      expect(c2.testingParentCtxValues.get("key")).toEqual(5);
      expect(c2.testingParentCtxValues.get("key2")).toEqual(6);
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
            message:
              "Failed to execute throwError(req-5) with args [] on invoke-leaf(invoke-test): Test error",
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
            message: expect.stringContaining("unknownMethod"),
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
