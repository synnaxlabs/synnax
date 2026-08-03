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

const MockSender = { send: vi.fn() };
const NOOP = alamos.Instrumentation.NOOP;

export const exampleProps = z.object({ x: z.number() });

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

/** Republishes "key" as `state.x * 10`, shadowing an ancestor that also publishes it. */
class ShadowingSetter extends aether.Composite<typeof exampleProps, {}, ExampleLeaf> {
  updatef = vi.fn();
  schema = exampleProps;

  afterUpdate(ctx: aether.Context): void {
    this.updatef(ctx);
    ctx.set("key", this.state.x * 10);
  }
}

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

  incrementSpy = vi.fn((n: number) => n + 1);
  greetSpy = vi.fn((args: { name: string }) => `Hello, ${args.name}!`);
  noArgsSpy = vi.fn(() => {});
  asyncMethodSpy = vi.fn(async (n: number) => n * 2);
  throwErrorSpy = vi.fn(() => {
    throw new Error("Test error");
  });

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

const make =
  <T extends aether.Component>(
    Ctor: new (props: aether.ComponentConstructorProps) => T,
    type: string,
  ) =>
  (key: string, parent: aether.Node | null = null): T =>
    new Ctor({ path: [key], type, sender: MockSender, instrumentation: NOOP, parent });

const createLeaf = make(ExampleLeaf, "leaf");
const createComposite = make(ExampleComposite, "composite");
const createContextSetter = make(ContextSetterComposite, "context");
const createSecondaryContextSetter = make(SecondaryContextSetter, "context");
const createConditionalReader = make(ConditionalReader, "conditional");
const createShadowingSetter = make(ShadowingSetter, "shadow");
const createInstanceSetter = make(InstanceSetter, "instance");
const createInvokeLeaf = make(InvokeLeaf, "invoke-leaf");
const createInvokeComposite = make(InvokeComposite, "invoke-composite");

const shouldNotCallCreate = (): aether.Component => {
  throw new Error("should not call create");
};

/** Issues a self/child state update with the standard `{ x }` state. `create` defaults to
 * {@link shouldNotCallCreate} for updates that should not spawn a child. */
const update = (
  c: aether.Component,
  path: string[],
  x: number,
  create: (parent: aether.Node) => aether.Component = shouldNotCallCreate,
): void => c._updateState({ path, state: { x }, type: "example", create });

describe("Aether Worker", () => {
  describe("AetherLeaf", () => {
    let leaf: ExampleLeaf;
    beforeEach(() => {
      leaf = createLeaf("test");
    });

    describe("internalUpdate", () => {
      it("should throw an error if the path is empty", () => {
        expect(() => update(leaf, [], 0)).toThrow(/empty path/);
        expect(leaf.updatef).toHaveBeenCalledTimes(0);
      });

      it("should throw an error if the path has a subpath", () => {
        expect(() => update(leaf, ["test", "dog"], 0)).toThrow(/subPath/);
        expect(leaf.updatef).toHaveBeenCalledTimes(0);
      });

      it("should throw an error if the path does not have the correct key", () => {
        expect(() => update(leaf, ["dog"], 0)).toThrow(/key/);
        expect(leaf.updatef).toHaveBeenCalledTimes(0);
      });

      it("should correctly internalUpdate the state", () => {
        update(leaf, ["test"], 2);
        expect(leaf.state).toEqual({ x: 2 });
      });

      it("should call the handleUpdate", () => {
        update(leaf, ["test"], 2);
        expect(leaf.updatef).toHaveBeenCalledTimes(1);
      });
    });

    describe("internalDelete", () => {
      it("should call the bound onDelete handler", () => {
        leaf._delete(["test"]);
        expect(leaf.deletef).toHaveBeenCalledTimes(1);
      });
    });

    describe("setState", () => {
      it("should communicate the state call to the main thread Sender", () => {
        update(leaf, ["test"], 2);
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
    beforeEach(() => {
      composite = createComposite("test");
    });

    describe("setState", () => {
      it("should set the state of the composite itself if the path has one element", () => {
        update(composite, ["test"], 2);
        expect(composite.state).toEqual({ x: 2 });
        expect(composite.updatef).toHaveBeenCalledTimes(1);
      });

      it("should create a new leaf if the path has more than one element and the leaf does not exist", () => {
        update(composite, ["test", "dog"], 2, () => createLeaf("dog"));
        expect(composite.children).toHaveLength(1);
        const c = composite.children[0];
        expect(c.key).toEqual("dog");
        expect(c.state).toEqual({ x: 2 });
        expect(c.updatef).toHaveBeenCalledTimes(1);
      });

      it("should set the state of the composite's leaf if the path has more than one element and the leaf exists", () => {
        update(composite, ["test", "dog"], 2, () => createLeaf("dog"));
        update(composite, ["test", "dog"], 3);
        expect(composite.children).toHaveLength(1);
        expect(composite.children[0].state).toEqual({ x: 3 });
      });

      it("should throw an error if the path is too deep and the child does not exist", () => {
        expect(() => update(composite, ["test", "dog", "cat"], 2)).toThrow(
          /attempting to create a new child/,
        );
      });
    });

    describe("internalDelete", () => {
      it("should remove a child from the list of children", () => {
        update(composite, ["test", "dog"], 2, () => createLeaf("dog"));
        expect(composite.children).toHaveLength(1);
        composite._delete(["test", "dog"]);
        expect(composite.children).toHaveLength(0);
      });

      it("should call the deletion hook on the child of a composite", () => {
        update(composite, ["test", "dog"], 2, () => createLeaf("dog"));
        const c = composite.children[0];
        composite._delete(["test", "dog"]);
        expect(c.deletef).toHaveBeenCalled();
      });
    });
  });

  describe("context propagation", () => {
    it("should correctly set a context value", () => {
      const v = createContextSetter("test");
      update(v, ["test"], 2);
      expect(v.testingChildCtxValues.get("key")).toEqual(2);
    });

    it("should correctly pass an initial context value to a leaf child", () => {
      const v = createContextSetter("test");
      update(v, ["test"], 2);
      update(v, ["test", "dog"], 3, (c) => createLeaf("dog", c));
      const c = v.children[0] as ExampleLeaf;
      expect(c.seen.key).toEqual(2);
      expect(c.testingChildCtxValues.size).toEqual(0);
    });

    it("should correctly update the context value in a child leaf", () => {
      const v = createContextSetter("test");
      update(v, ["test"], 2);
      update(v, ["test", "dog"], 3, (c) => createLeaf("dog", c));
      const c = v.children[0] as ExampleLeaf;
      update(v, ["test"], 4);
      expect(v.testingChildCtxValues.get("key")).toEqual(4);
      expect(c.seen.key).toEqual(4);
      expect(c.testingChildCtxValues.size).toEqual(0);
    });

    it("should correctly separate individual contexts", () => {
      const v = createComposite("test");
      update(v, ["test"], 2);
      update(v, ["test", "dog"], 3, (c) => createContextSetter("dog", c));
      update(v, ["test", "cat"], 4, (c) => createContextSetter("cat", c));
      expect(v.children).toHaveLength(2);
      const [c1, c2] = v.children;
      expect(c1.testingChildCtxValues.size).toEqual(1);
      expect(c2.testingChildCtxValues.size).toEqual(1);
      expect(c1.testingChildCtxValues.get("key")).toEqual(3);
      expect(c2.testingChildCtxValues.get("key")).toEqual(4);
    });

    it("should correctly initialize contexts with a nested leaf", () => {
      const v = createContextSetter("first");
      update(v, ["first"], 2);
      expect(v.testingChildCtxValues.size).toEqual(1);
      update(v, ["first", "second"], 3, (c) =>
        createSecondaryContextSetter("second", c),
      );
      const c1 = v.children[0] as SecondaryContextSetter;
      expect(c1.seen.key).toEqual(2);
      expect(c1.testingChildCtxValues.get("key2")).toEqual(3);
      update(v, ["first", "second", "third"], 4, (c) => createLeaf("third", c));
      const c2 = (v.children[0] as SecondaryContextSetter).children[0];
      expect(c2.testingChildCtxValues.size).toEqual(0);
      expect(c2.seen.key).toEqual(2);
      expect(c2.seen.key2).toEqual(3);
    });

    it("should correctly update contexts with a nested leaf", () => {
      const v = createContextSetter("first");
      update(v, ["first"], 2);
      update(v, ["first", "second"], 3, (c) =>
        createSecondaryContextSetter("second", c),
      );
      const c1 = v.children[0] as SecondaryContextSetter;
      update(v, ["first", "second", "third"], 4, (c) => createLeaf("third", c));
      const c2 = (v.children[0] as SecondaryContextSetter).children[0];
      update(v, ["first"], 5);
      expect(c1.seen.key).toEqual(5);
      expect(c1.testingChildCtxValues.get("key2")).toEqual(6);
      expect(c2.seen.key).toEqual(5);
      expect(c2.seen.key2).toEqual(6);
    });

    it("re-runs only the subscribers of a changed key, not the whole subtree", () => {
      const root = createContextSetter("root");
      update(root, ["root"], 1);
      // reader subscribes to "key"; ignorer never reads context.
      let reader!: ExampleLeaf;
      let ignorer!: ExampleComposite;
      update(root, ["root", "reader"], 0, (c) => (reader = createLeaf("reader", c)));
      update(
        root,
        ["root", "ignorer"],
        0,
        (c) => (ignorer = createComposite("ignorer", c)),
      );
      reader.updatef.mockClear();
      ignorer.updatef.mockClear();
      update(root, ["root"], 2);
      expect(reader.updatef).toHaveBeenCalledTimes(1);
      expect(reader.seen.key).toEqual(2);
      expect(ignorer.updatef).not.toHaveBeenCalled();
    });

    it("subscribes and unsubscribes as a conditional read appears and disappears", () => {
      const root = createContextSetter("root");
      update(root, ["root"], 1);
      // ConditionalReader reads "key" only when state.x > 0.
      let cond!: ConditionalReader;
      update(
        root,
        ["root", "cond"],
        1,
        (c) => (cond = createConditionalReader("cond", c)),
      );
      expect(cond.seen).toEqual(1);

      // Stop reading "key" (x -> 0); this re-runs cond and unsubscribes it.
      update(root, ["root", "cond"], 0);
      cond.updatef.mockClear();
      update(root, ["root"], 5);
      expect(cond.updatef).not.toHaveBeenCalled();

      // Resume reading "key" (x -> 1); now changes to "key" re-run it again.
      update(root, ["root", "cond"], 1);
      cond.updatef.mockClear();
      update(root, ["root"], 9);
      expect(cond.updatef).toHaveBeenCalledTimes(1);
      expect(cond.seen).toEqual(9);
    });

    it("binds a consumer to the nearest provider when a key is shadowed", () => {
      const root = createContextSetter("root");
      update(root, ["root"], 1);
      // mid republishes "key" (x * 10), shadowing root for its descendants.
      let mid!: ShadowingSetter;
      update(root, ["root", "mid"], 2, (c) => (mid = createShadowingSetter("mid", c)));
      update(root, ["root", "mid", "leaf"], 0, (c) => createLeaf("leaf", c));
      const leaf = mid.children[0];
      expect(leaf.seen.key).toEqual(20);

      // Changing the far provider (root) does NOT re-run the shadowed leaf.
      leaf.updatef.mockClear();
      update(root, ["root"], 7);
      expect(leaf.updatef).not.toHaveBeenCalled();
      expect(leaf.seen.key).toEqual(20);

      // Changing the near provider (mid) does re-run it.
      leaf.updatef.mockClear();
      update(root, ["root", "mid"], 3);
      expect(leaf.updatef).toHaveBeenCalledTimes(1);
      expect(leaf.seen.key).toEqual(30);
    });

    it("cascades to subscribers when a provider re-sets the same value reference", () => {
      const root = createInstanceSetter("root");
      update(root, ["root"], 1);
      update(root, ["root", "leaf"], 0, (c) => createLeaf("leaf", c));
      const leaf = root.children[0];
      leaf.updatef.mockClear();
      // A self-update re-runs root.afterUpdate, which re-sets the SAME value with
      // trigger=true (the canvas pattern). Subscribers must still re-run.
      update(root, ["root"], 2);
      expect(leaf.updatef).toHaveBeenCalledTimes(1);
    });

    it("unsubscribes a deleted consumer so it is not re-run", () => {
      const root = createContextSetter("root");
      update(root, ["root"], 1);
      update(root, ["root", "leaf"], 0, (c) => createLeaf("leaf", c));
      const leaf = root.children[0] as ExampleLeaf;
      root._delete(["root", "leaf"]);
      leaf.updatef.mockClear();
      update(root, ["root"], 2);
      expect(leaf.updatef).not.toHaveBeenCalled();
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

        await scheduler.flushTaskQueue();

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

// Property/oracle test for context propagation. The incremental engine (re-run only the
// subscribers of a changed key, cascading) is checked against a trivially-correct oracle
// — a full top-down recompute of the whole tree. For random trees and random update /
// delete sequences, every live node's resolved context must match the recompute; any
// missed propagation (a stale node) makes the two disagree.

const stateZ = z.object({ x: z.number(), id: z.string() });

const KEYS = ["a", "b", "c", "d"] as const;
type Key = (typeof KEYS)[number];
const KEY_WEIGHT: Record<Key, number> = { a: 1, b: 10, c: 100, d: 1000 };

interface Behavior {
  /** Keys always read. */
  reads: Key[];
  /** Keys read only when state.x is even — exercises dynamic subscribe/unsubscribe. */
  condReads: Key[];
  /** Keys published every afterUpdate (empty for leaves). Publishing is unconditional,
   * matching the real-world invariant that providers establish keys at mount. */
  publishes: Key[];
}

interface Observation {
  reads: Record<string, number>;
  published: Record<string, number>;
}

// Per-case shared state, re-pointed each iteration. Components read these module
// bindings at afterUpdate time, so reassigning them between cases is safe.
let behaviors = new Map<string, Behavior>();
let recorded = new Map<string, Observation>();

const activeReadKeys = (b: Behavior, x: number): Key[] => [
  ...new Set<Key>([...b.reads, ...(x % 2 === 0 ? b.condReads : [])]),
];

const publishedValue = (x: number, key: Key, reads: Record<string, number>): number => {
  let sum = 0;
  for (const v of Object.values(reads)) sum += v;
  return x + KEY_WEIGHT[key] + sum;
};

// Shared afterUpdate body for both the leaf and composite test components.
const runNode = (state: { x: number; id: string }, ctx: aether.Context): void => {
  const b = behaviors.get(state.id);
  if (b == null) throw new Error(`missing behavior for ${state.id}`);
  const reads: Record<string, number> = {};
  for (const k of activeReadKeys(b, state.x)) {
    const v = ctx.getOptional<number>(k);
    if (v != null) reads[k] = v;
  }
  const published: Record<string, number> = {};
  for (const k of b.publishes) {
    const v = publishedValue(state.x, k, reads);
    published[k] = v;
    ctx.set(k, v);
  }
  recorded.set(state.id, { reads, published });
};

class PropComposite extends aether.Composite<typeof stateZ> {
  schema = stateZ;
  afterUpdate(ctx: aether.Context): void {
    runNode(this.state, ctx);
  }
}

class PropLeaf extends aether.Leaf<typeof stateZ> {
  schema = stateZ;
  afterUpdate(ctx: aether.Context): void {
    runNode(this.state, ctx);
  }
}

interface TreeNode {
  id: string;
  key: string;
  fullPath: string[];
  isLeaf: boolean;
  parentId: string | null;
}

// Deterministic PRNG (mulberry32) so each case is reproducible from its seed.
const rng = (seed: number) => () => {
  seed = (seed + 0x6d2b79f5) | 0;
  let t = Math.imul(seed ^ (seed >>> 15), 1 | seed);
  t ^= t + Math.imul(t ^ (t >>> 7), 61 | t);
  return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
};

const pick = <T>(rand: () => number, arr: T[]): T =>
  arr[Math.floor(rand() * arr.length)];

const pickSubset = (rand: () => number, max: number): Key[] =>
  KEYS.filter(() => rand() < 0.4).slice(0, max);

const genTree = (rand: () => number): TreeNode[] => {
  const root: TreeNode = {
    id: "n0",
    key: "n0",
    fullPath: ["n0"],
    isLeaf: false,
    parentId: null,
  };
  const nodes: TreeNode[] = [root];
  const composites: TreeNode[] = [root];
  const count = 5 + Math.floor(rand() * 16);
  for (let i = 1; i < count; i++) {
    const parent = pick(rand, composites);
    const isLeaf = rand() < 0.5;
    const key = `n${i}`;
    const node: TreeNode = {
      id: key,
      key,
      fullPath: [...parent.fullPath, key],
      isLeaf,
      parentId: parent.id,
    };
    nodes.push(node);
    if (!isLeaf) composites.push(node);
  }
  for (const node of nodes)
    behaviors.set(node.id, {
      reads: pickSubset(rand, 2),
      condReads: pickSubset(rand, 1),
      publishes: node.isLeaf ? [] : pickSubset(rand, 2),
    });
  return nodes;
};

// Brute-force, obviously-correct reference: recompute every live node top-down.
const oracle = (
  nodes: TreeNode[],
  live: Set<string>,
  states: Map<string, number>,
): Map<string, Observation> => {
  const byId = new Map(nodes.map((n) => [n.id, n]));
  const liveNodes = nodes
    .filter((n) => live.has(n.id))
    .sort((a, b) => a.fullPath.length - b.fullPath.length);
  const pub = new Map<string, Record<string, number>>();
  const out = new Map<string, Observation>();
  for (const node of liveNodes) {
    const x = states.get(node.id) as number;
    const b = behaviors.get(node.id) as Behavior;
    const reads: Record<string, number> = {};
    for (const k of activeReadKeys(b, x)) {
      let anc = node.parentId == null ? null : byId.get(node.parentId);
      while (anc != null) {
        const v = pub.get(anc.id)?.[k];
        if (v !== undefined) {
          reads[k] = v;
          break;
        }
        anc = anc.parentId == null ? null : byId.get(anc.parentId);
      }
    }
    const published: Record<string, number> = {};
    for (const k of b.publishes) published[k] = publishedValue(x, k, reads);
    pub.set(node.id, published);
    out.set(node.id, { reads, published });
  }
  return out;
};

const mk = (node: TreeNode, parent: aether.Node | null): aether.Component => {
  const props = {
    path: [node.key],
    type: node.isLeaf ? "leaf" : "node",
    sender: MockSender,
    instrumentation: NOOP,
    parent,
  };
  return node.isLeaf ? new PropLeaf(props) : new PropComposite(props);
};

describe("aether context propagation (property/oracle)", () => {
  it("keeps every node's resolved context equal to a full recompute", () => {
    const CASES = 300;
    for (let seed = 0; seed < CASES; seed++) {
      const rand = rng(seed + 1);
      behaviors = new Map();
      recorded = new Map();

      const nodes = genTree(rand);
      const byId = new Map(nodes.map((n) => [n.id, n]));
      const states = new Map<string, number>();
      const randX = () => Math.floor(rand() * 6);

      // Build and drive the tree by routing through the root, exactly as production does.
      const root = mk(nodes[0], null);
      const apply = (
        node: TreeNode,
        x: number,
        create: (parent: aether.Node) => aether.Component = shouldNotCallCreate,
      ): void => {
        states.set(node.id, x);
        root._updateState({
          path: node.fullPath,
          state: { x, id: node.id },
          type: node.isLeaf ? "leaf" : "node",
          create,
        });
      };
      apply(nodes[0], randX());
      for (const node of nodes.slice(1)) apply(node, randX(), (p) => mk(node, p));

      // Drive a random sequence of state updates and deletes.
      const live = new Set(nodes.map((n) => n.id));
      const ops = 10 + Math.floor(rand() * 11);
      const liveList = () => [...live].map((id) => byId.get(id) as TreeNode);
      for (let m = 0; m < ops; m++) {
        const deletable = liveList().filter((n) => n.id !== "n0");
        if (rand() < 0.2 && deletable.length > 0) {
          const target = pick(rand, deletable);
          root._delete(target.fullPath);
          // Remove the deleted node and its whole subtree from the live set.
          for (const n of nodes)
            if (target.fullPath.every((seg, i) => n.fullPath[i] === seg))
              live.delete(n.id);
        } else apply(pick(rand, liveList()), randX());
      }

      const expected = oracle(nodes, live, states);
      for (const id of live)
        expect(
          recorded.get(id),
          `seed ${seed}, node ${id}: incremental propagation diverged from full recompute`,
        ).toEqual(expected.get(id));
    }
  });
});
