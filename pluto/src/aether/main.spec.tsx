// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createMockWorkers } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { type FC, type PropsWithChildren, useRef } from "react";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Aether } from "@/aether";
import { aether } from "@/aether/aether";
import { type AetherMessage, type MainMessage } from "@/aether/message";

export const exampleProps = z.object({
  x: z.number(),
});

class ExampleLeaf extends aether.Leaf<typeof exampleProps> {
  static readonly TYPE = "ExampleLeaf";
  updatef = vi.fn();
  deletef = vi.fn();

  schema = exampleProps;

  afterUpdate(): void {
    this.updatef();
  }

  afterDelete(): void {
    this.deletef();
  }
}

class ExampleComposite extends aether.Composite<typeof exampleProps, ExampleLeaf> {
  updatef = vi.fn();
  deletef = vi.fn();

  static readonly TYPE = "ExampleComposite";

  schema = exampleProps;

  afterUpdate(): void {
    this.updatef();
  }

  afterDelete(): void {
    this.deletef();
  }
}

const rpcMethodsSchema = {
  fireAndForget: z.function({ output: z.void() }),
  getValue: z.function({ output: z.number() }),
  throwError: z.function({ output: z.number() }),
  neverResponds: z.function({ output: z.number() }),
  updateState: z.function({ input: z.tuple([z.number()]), output: z.void() }),
} satisfies aether.MethodsSchema;

class RPCLeaf
  extends aether.Leaf<typeof exampleProps, {}, typeof rpcMethodsSchema>
  implements aether.HandlersFromSchema<typeof rpcMethodsSchema>
{
  static readonly TYPE = "RPCLeaf";
  schema = exampleProps;
  methods = rpcMethodsSchema;
  fireAndForgetSpy = vi.fn();
  getValueSpy = vi.fn(() => 42);
  throwErrorSpy = vi.fn(() => {
    throw new Error("Test error");
  });
  neverRespondsSpy = vi.fn(() => new Promise<number>(() => {}));
  fireAndForget(): void {
    this.fireAndForgetSpy();
  }
  getValue(): number {
    return this.getValueSpy();
  }
  throwError(): number {
    return this.throwErrorSpy();
  }
  neverResponds(): Promise<number> {
    return this.neverRespondsSpy();
  }
  updateState(x: number): void {
    this.setState({ x });
  }
  afterUpdate(): void {}
  afterDelete(): void {}
}

const REGISTRY: aether.ComponentRegistry = {
  [ExampleLeaf.TYPE]: ExampleLeaf,
  [ExampleComposite.TYPE]: ExampleComposite,
  [RPCLeaf.TYPE]: RPCLeaf,
};

const newProvider = async (): Promise<[FC<PropsWithChildren>, aether.Root]> => {
  const [a, b] = createMockWorkers();
  const root = aether.render({ comms: a.route("vis"), registry: REGISTRY });
  const worker = b.route<MainMessage, AetherMessage>("vis");
  return [
    (props: PropsWithChildren) => (
      <Aether.Provider worker={worker} workerKey="vis" {...props} />
    ),
    root,
  ];
};

describe("Aether Main", () => {
  describe("leaf", () => {
    it("should set the initial state correctly", async () => {
      const [Provider, root] = await newProvider();
      const ExampleLeafC = () => {
        Aether.use({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
        });
        return null;
      };
      render(
        <Provider>
          <ExampleLeafC />
        </Provider>,
      );
      await expect.poll(async () => root.children.length === 1).toBe(true);
      const first = root.children[0] as ExampleLeaf;
      expect(first.type).toBe(ExampleLeaf.TYPE);
      expect(first.state).toEqual({ x: 0 });
    });
    it("should update the state on a call to setState", async () => {
      const [Provider, root] = await newProvider();
      const ExampleLeafC = () => {
        const [, , setState] = Aether.use({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
        });
        const set = useRef(false);
        if (!set.current) {
          setState({ x: 1 });
          set.current = true;
        }
        return null;
      };
      render(
        <Provider>
          <ExampleLeafC />
        </Provider>,
      );
      await expect.poll(async () => root.children.length === 1).toBe(true);
      const first = root.children[0] as ExampleLeaf;
      expect(first.type).toBe(ExampleLeaf.TYPE);
      expect(first.state).toEqual({ x: 1 });
    });
    it("should trigger afterDelete on unmount", async () => {
      const [Provider, root] = await newProvider();
      const ExampleLeafC = () => {
        Aether.use({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
        });
        return null;
      };
      const { unmount } = render(
        <Provider>
          <ExampleLeafC />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      const leaf = root.children[0] as ExampleLeaf;
      expect(leaf.deletef).not.toHaveBeenCalled();
      unmount();
      await expect.poll(() => leaf.deletef.mock.calls.length > 0).toBe(true);
    });
  });
  describe("RPC", () => {
    it("should invoke fire-and-forget method on worker", async () => {
      const [Provider, root] = await newProvider();
      const RPCLeafC = () => {
        const [, , , methods] = Aether.use({
          type: RPCLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          methods: rpcMethodsSchema,
        });
        const called = useRef(false);
        if (!called.current) {
          called.current = true;
          methods.fireAndForget();
        }
        return null;
      };
      render(
        <Provider>
          <RPCLeafC />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      const leaf = root.children[0] as RPCLeaf;
      await expect.poll(() => leaf.fireAndForgetSpy.mock.calls.length > 0).toBe(true);
    });
    it("should resolve async RPC with worker return value", async () => {
      const [Provider, root] = await newProvider();
      let result: number | null = null;
      const RPCLeafC = () => {
        const [, , , methods] = Aether.use({
          type: RPCLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          methods: rpcMethodsSchema,
        });
        const called = useRef(false);
        if (!called.current) {
          called.current = true;
          void methods.getValue().then((v) => {
            result = v;
          });
        }
        return null;
      };
      render(
        <Provider>
          <RPCLeafC />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      await expect.poll(() => result !== null).toBe(true);
      expect(result).toBe(42);
    });
    it("should reject async RPC when worker method throws", async () => {
      const [Provider, root] = await newProvider();
      const captured: { error: Error | null } = { error: null };
      const RPCLeafC = () => {
        const [, , , methods] = Aether.use({
          type: RPCLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          methods: rpcMethodsSchema,
        });
        const called = useRef(false);
        if (!called.current) {
          called.current = true;
          void methods.throwError().catch((e: Error) => {
            captured.error = e;
          });
        }
        return null;
      };
      render(
        <Provider>
          <RPCLeafC />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      await expect.poll(() => captured.error !== null).toBe(true);
      expect(captured.error?.message).toContain("Test error");
    });
    it("should reject async RPC on timeout", async () => {
      vi.useFakeTimers();
      const [Provider, root] = await newProvider();
      const captured: { error: Error | null } = { error: null };
      const RPCLeafC = () => {
        const [, , , methods] = Aether.use({
          type: RPCLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          methods: rpcMethodsSchema,
        });
        const called = useRef(false);
        if (!called.current) {
          called.current = true;
          void methods.neverResponds().catch((e: Error) => {
            captured.error = e;
          });
        }
        return null;
      };
      render(
        <Provider>
          <RPCLeafC />
        </Provider>,
      );
      await vi.waitFor(() => expect(root.children.length).toBe(1));
      expect(captured.error).toBeNull();
      await vi.advanceTimersByTimeAsync(5001);
      expect(captured.error).not.toBeNull();
      expect(captured.error?.name).toBe("TimeoutError");
      vi.useRealTimers();
    });
    it("should abort pending RPC on component unmount", async () => {
      const [Provider, root] = await newProvider();
      const captured: { error: Error | null } = { error: null };
      const RPCLeafC = () => {
        const [, , , methods] = Aether.use({
          type: RPCLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          methods: rpcMethodsSchema,
        });
        const called = useRef(false);
        if (!called.current) {
          called.current = true;
          void methods.neverResponds().catch((e: Error) => {
            captured.error = e;
          });
        }
        return null;
      };
      const { unmount } = render(
        <Provider>
          <RPCLeafC />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      expect(captured.error).toBeNull();
      unmount();
      await expect.poll(() => captured.error !== null).toBe(true);
      expect(captured.error?.message).toBe("Component deleted");
    });
  });
  describe("use hook", () => {
    it("should call onAetherChange when worker updates state", async () => {
      const [Provider, root] = await newProvider();
      const onAetherChange = vi.fn();
      const RPCLeafC = () => {
        const [, , , methods] = Aether.use({
          type: RPCLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          methods: rpcMethodsSchema,
          onAetherChange,
        });
        const called = useRef(false);
        if (!called.current) {
          called.current = true;
          methods.updateState(99);
        }
        return null;
      };
      render(
        <Provider>
          <RPCLeafC />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      await expect.poll(() => onAetherChange.mock.calls.length > 0).toBe(true);
      expect(onAetherChange).toHaveBeenCalledWith({ x: 99 });
    });
    it("should pass current state to functional setState", async () => {
      const [Provider, root] = await newProvider();
      const ExampleLeafC = () => {
        const [, state, setState] = Aether.use({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 10 },
        });
        const called = useRef(false);
        if (!called.current && state.x === 10) {
          called.current = true;
          setState((prev) => ({ x: prev.x + 5 }));
        }
        return null;
      };
      render(
        <Provider>
          <ExampleLeafC />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      const leaf = root.children[0] as ExampleLeaf;
      await expect.poll(() => leaf.state.x === 15).toBe(true);
    });
  });
  describe("useUnidirectional", () => {
    it("should update worker when state prop changes", async () => {
      const [Provider, root] = await newProvider();
      const UnidirectionalLeaf = ({ x }: { x: number }) => {
        Aether.useUnidirectional({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          state: { x },
        });
        return null;
      };
      const { rerender } = render(
        <Provider>
          <UnidirectionalLeaf x={1} />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      const leaf = root.children[0] as ExampleLeaf;
      expect(leaf.state.x).toBe(1);
      rerender(
        <Provider>
          <UnidirectionalLeaf x={42} />
        </Provider>,
      );
      await expect.poll(() => leaf.state.x === 42).toBe(true);
    });
    it("should not re-send when state is deeply equal", async () => {
      const [Provider, root] = await newProvider();
      const UnidirectionalLeaf = ({ x }: { x: number }) => {
        Aether.useUnidirectional({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          state: { x },
        });
        return null;
      };
      const { rerender } = render(
        <Provider>
          <UnidirectionalLeaf x={5} />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      const leaf = root.children[0] as ExampleLeaf;
      await expect.poll(() => leaf.updatef.mock.calls.length >= 1).toBe(true);
      const initialCount = leaf.updatef.mock.calls.length;
      rerender(
        <Provider>
          <UnidirectionalLeaf x={5} />
        </Provider>,
      );
      await new Promise((r) => setTimeout(r, 50));
      expect(leaf.updatef.mock.calls.length).toBe(initialCount);
    });
  });
  describe("useLifecycle", () => {
    it("should trigger delete on unmount", async () => {
      const [Provider, root] = await newProvider();
      const LifecycleLeaf = () => {
        Aether.useLifecycle({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
        });
        return null;
      };
      const { unmount } = render(
        <Provider>
          <LifecycleLeaf />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      const leaf = root.children[0] as ExampleLeaf;
      expect(leaf.deletef).not.toHaveBeenCalled();
      unmount();
      await expect.poll(() => leaf.deletef.mock.calls.length > 0).toBe(true);
    });
  });
  describe("Composite", () => {
    it("should nest children under parent path", async () => {
      const [Provider, root] = await newProvider();
      const Parent = () => {
        const [{ path }] = Aether.use({
          type: ExampleComposite.TYPE,
          schema: exampleProps,
          initialState: { x: 1 },
        });
        return (
          <Aether.Composite path={path}>
            <Child />
          </Aether.Composite>
        );
      };
      const Child = () => {
        Aether.use({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 2 },
        });
        return null;
      };
      render(
        <Provider>
          <Parent />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      const composite = root.children[0] as ExampleComposite;
      expect(composite.type).toBe(ExampleComposite.TYPE);
      await expect.poll(() => composite.children.length === 1).toBe(true);
      const leaf = composite.children[0] as ExampleLeaf;
      expect(leaf.type).toBe(ExampleLeaf.TYPE);
      expect(leaf.state.x).toBe(2);
    });
  });
});
