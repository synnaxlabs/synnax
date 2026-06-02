// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { render, waitFor } from "@testing-library/react";
import {
  Component,
  type FC,
  type PropsWithChildren,
  type ReactNode,
  StrictMode,
  useEffect,
  useRef,
} from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Aether } from "@/aether";
import { aether } from "@/aether/aether";

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

const invokeMethodsSchema = {
  fireAndForget: z.function({ output: z.void() }),
  getValue: z.function({ output: z.number() }),
  throwError: z.function({ output: z.number() }),
  neverResponds: z.function({ output: z.number() }),
  updateState: z.function({ input: z.tuple([z.number()]), output: z.void() }),
  echo: z.function({
    input: z.tuple([z.number(), z.string()]),
    output: z.tuple([z.number(), z.string()]),
  }),
  slowEcho: z.function({ input: z.tuple([z.number()]), output: z.number() }),
} satisfies aether.MethodsSchema;

class InvokeLeaf
  extends aether.Leaf<typeof exampleProps, {}, typeof invokeMethodsSchema>
  implements aether.HandlersFromSchema<typeof invokeMethodsSchema>
{
  static readonly TYPE = "InvokeLeaf";
  schema = exampleProps;
  methods = invokeMethodsSchema;
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
  echoSpy = vi.fn((n: number, s: string): [number, string] => [n, s]);
  echo(n: number, s: string): [number, string] {
    return this.echoSpy(n, s);
  }
  slowEchoResolvers: Array<() => void> = [];
  async slowEcho(n: number): Promise<number> {
    return await new Promise<number>((resolve) => {
      this.slowEchoResolvers.push(() => resolve(n));
    });
  }
  afterUpdate(): void {}
  afterDelete(): void {}
}

const multiProps = z.object({
  x: z.number(),
  y: z.string(),
});

class MultiLeaf extends aether.Leaf<typeof multiProps> {
  static readonly TYPE = "MultiLeaf";
  schema = multiProps;
  afterUpdate(): void {}
  afterDelete(): void {}
}

class EchoOnUpdateLeaf extends aether.Leaf<typeof exampleProps> {
  static readonly TYPE = "EchoOnUpdateLeaf";
  schema = exampleProps;
  private echoed = false;
  afterUpdate(): void {
    if (this.echoed) return;
    this.echoed = true;
    // Defer the push so it lands outside the synchronous mock-worker round-
    // trip — the property the pre-mount state-update path needs to test. A
    // real Worker.postMessage delivers on a macrotask; queueMicrotask is a
    // microtask, but either satisfies "outside the current render" which is
    // what this fixture exists to exercise.
    queueMicrotask(() => this.setState({ x: this.state.x + 1 }));
  }
  afterDelete(): void {}
}

class ErrorOnUpdateLeaf extends aether.Leaf<typeof exampleProps> {
  static readonly TYPE = "ErrorOnUpdateLeaf";
  schema = exampleProps;
  afterUpdate(): void {
    throw new Error("ErrorOnUpdateLeaf afterUpdate failure");
  }
  afterDelete(): void {}
}

const REGISTRY: aether.ComponentRegistry = {
  [ExampleLeaf.TYPE]: ExampleLeaf,
  [ExampleComposite.TYPE]: ExampleComposite,
  [InvokeLeaf.TYPE]: InvokeLeaf,
  [MultiLeaf.TYPE]: MultiLeaf,
  [EchoOnUpdateLeaf.TYPE]: EchoOnUpdateLeaf,
  [ErrorOnUpdateLeaf.TYPE]: ErrorOnUpdateLeaf,
};

type TestProviderProps = PropsWithChildren<
  Partial<Omit<Aether.ProviderProps, "worker">>
>;

const newProvider = async (): Promise<[FC<TestProviderProps>, aether.Root]> => {
  const [workerSide, mainSide] = aether.createMockPair();
  const root = aether.render({ worker: workerSide, registry: REGISTRY });
  return [
    (props: TestProviderProps) => <Aether.Provider worker={mainSide} {...props} />,
    root,
  ];
};

class FakeWorker {
  onmessage: ((e: MessageEvent) => void) | null = null;
  onerror: ((e: ErrorEvent) => void) | null = null;
  onmessageerror: ((e: MessageEvent) => void) | null = null;
  postMessage = vi.fn();
  terminate = vi.fn();
  constructor(public url: string | URL) {}
}

interface CapturedError {
  current: Error | null;
}

class ErrorBoundary extends Component<
  PropsWithChildren<{ captured: CapturedError; fallback?: ReactNode }>,
  { hasError: boolean }
> {
  state = { hasError: false };
  static getDerivedStateFromError() {
    return { hasError: true };
  }
  componentDidCatch(error: Error): void {
    this.props.captured.current = error;
  }
  render(): ReactNode {
    if (this.state.hasError) return this.props.fallback ?? null;
    return this.props.children;
  }
}

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
  describe("invoke", () => {
    it("should invoke fire-and-forget method on worker", async () => {
      const [Provider, root] = await newProvider();
      const InvokeLeafC = () => {
        const [, , , methods] = Aether.use({
          type: InvokeLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          methods: invokeMethodsSchema,
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
          <InvokeLeafC />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      const leaf = root.children[0] as InvokeLeaf;
      await expect.poll(() => leaf.fireAndForgetSpy.mock.calls.length > 0).toBe(true);
    });
    it("should resolve async invoke with worker return value", async () => {
      const [Provider, root] = await newProvider();
      let result: number | null = null;
      const InvokeLeafC = () => {
        const [, , , methods] = Aether.use({
          type: InvokeLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          methods: invokeMethodsSchema,
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
          <InvokeLeafC />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      await expect.poll(() => result !== null).toBe(true);
      expect(result).toBe(42);
    });
    it("should reject async invoke when worker method throws", async () => {
      const [Provider, root] = await newProvider();
      const captured: { error: Error | null } = { error: null };
      const InvokeLeafC = () => {
        const [, , , methods] = Aether.use({
          type: InvokeLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          methods: invokeMethodsSchema,
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
          <InvokeLeafC />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      await expect.poll(() => captured.error !== null).toBe(true);
      expect(captured.error?.message).toContain("Test error");
    });
    it("should reject async invoke on timeout", async () => {
      const [Provider, root] = await newProvider();
      const captured: { error: Error | null } = { error: null };
      const InvokeLeafC = () => {
        const [, , , methods] = Aether.use({
          type: InvokeLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          methods: invokeMethodsSchema,
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
        <Provider invokeTimeout={TimeSpan.milliseconds(50)}>
          <InvokeLeafC />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      expect(captured.error).toBeNull();
      await waitFor(() => expect(captured.error).not.toBeNull());
      expect(captured.error?.name).toBe("TimeoutError");
    });
    it("should abort pending invoke on component unmount", async () => {
      const [Provider, root] = await newProvider();
      const captured: { error: Error | null } = { error: null };
      const InvokeLeafC = () => {
        const [, , , methods] = Aether.use({
          type: InvokeLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          methods: invokeMethodsSchema,
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
          <InvokeLeafC />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      expect(captured.error).toBeNull();
      unmount();
      await expect.poll(() => captured.error !== null).toBe(true);
      expect(captured.error?.message).toBe("Component deleted");
    });
    it("should forward multiple positional args spread to the worker handler", async () => {
      const [Provider, root] = await newProvider();
      const captured: { result: [number, string] | null } = { result: null };
      const C = () => {
        const [, , , methods] = Aether.use({
          type: InvokeLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          methods: invokeMethodsSchema,
        });
        useEffect(() => {
          void methods.echo(7, "hello").then((r) => {
            captured.result = r;
          });
        }, [methods]);
        return null;
      };
      render(
        <Provider>
          <C />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      const leaf = root.children[0] as InvokeLeaf;
      await expect.poll(() => leaf.echoSpy.mock.calls.length > 0).toBe(true);
      expect(leaf.echoSpy.mock.calls[0]).toEqual([7, "hello"]);
      await expect.poll(() => captured.result !== null).toBe(true);
      expect(captured.result).toEqual([7, "hello"]);
    });
    it("should resolve multiple concurrent invokes to their own callers", async () => {
      const [Provider, root] = await newProvider();
      const results: number[] = [];
      const C = () => {
        const [, , , methods] = Aether.use({
          type: InvokeLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          methods: invokeMethodsSchema,
        });
        useEffect(() => {
          void methods.slowEcho(1).then((r) => results.push(r));
          void methods.slowEcho(2).then((r) => results.push(r));
          void methods.slowEcho(3).then((r) => results.push(r));
        }, [methods]);
        return null;
      };
      render(
        <Provider>
          <C />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      const leaf = root.children[0] as InvokeLeaf;
      await expect.poll(() => leaf.slowEchoResolvers.length === 3).toBe(true);
      // Resolve out of order to confirm correlation, not ordering, matters.
      leaf.slowEchoResolvers[2]();
      leaf.slowEchoResolvers[0]();
      leaf.slowEchoResolvers[1]();
      await expect.poll(() => results.length === 3).toBe(true);
      expect(results.sort()).toEqual([1, 2, 3]);
    });
    it("should invoke fire-and-forget methods in submission order", async () => {
      const [Provider, root] = await newProvider();
      const C = () => {
        const [, , , methods] = Aether.use({
          type: InvokeLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          methods: invokeMethodsSchema,
        });
        useEffect(() => {
          methods.fireAndForget();
          methods.fireAndForget();
          methods.fireAndForget();
        }, [methods]);
        return null;
      };
      render(
        <Provider>
          <C />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      const leaf = root.children[0] as InvokeLeaf;
      await expect.poll(() => leaf.fireAndForgetSpy.mock.calls.length === 3).toBe(true);
    });
  });
  describe("use hook", () => {
    it("should call onAetherChange when worker updates state", async () => {
      const [Provider, root] = await newProvider();
      const onAetherChange = vi.fn();
      const InvokeLeafC = () => {
        const [, , , methods] = Aether.use({
          type: InvokeLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          methods: invokeMethodsSchema,
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
          <InvokeLeafC />
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
    it("should propagate worker-pushed state to the main thread state", async () => {
      const [Provider] = await newProvider();
      const observed: { x: number | null } = { x: null };
      const C = () => {
        const [, state, , methods] = Aether.use({
          type: InvokeLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          methods: invokeMethodsSchema,
        });
        observed.x = state.x;
        const called = useRef(false);
        if (!called.current) {
          called.current = true;
          methods.updateState(42);
        }
        return null;
      };
      render(
        <Provider>
          <C />
        </Provider>,
      );
      await expect.poll(() => observed.x === 42).toBe(true);
    });
    it("should propagate multiple sequential worker pushes in order", async () => {
      const [Provider] = await newProvider();
      const observed: { x: number | null } = { x: null };
      const C = () => {
        const [, state, , methods] = Aether.use({
          type: InvokeLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          methods: invokeMethodsSchema,
        });
        observed.x = state.x;
        const called = useRef(false);
        if (!called.current) {
          called.current = true;
          methods.updateState(1);
          methods.updateState(2);
          methods.updateState(3);
        }
        return null;
      };
      render(
        <Provider>
          <C />
        </Provider>,
      );
      await expect.poll(() => observed.x === 3).toBe(true);
    });
    it("should keep setState identity stable across re-renders", async () => {
      const [Provider, root] = await newProvider();
      const captured: unknown[] = [];
      const C = ({ trigger }: { trigger: number }) => {
        const [, , setState] = Aether.use({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
        });
        captured.push(setState);
        return <span>{trigger}</span>;
      };
      const { rerender } = render(
        <Provider>
          <C trigger={1} />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      rerender(
        <Provider>
          <C trigger={2} />
        </Provider>,
      );
      rerender(
        <Provider>
          <C trigger={3} />
        </Provider>,
      );
      expect(captured.length).toBeGreaterThanOrEqual(3);
      expect(captured[0]).toBe(captured[captured.length - 1]);
      for (let i = 1; i < captured.length; i++) expect(captured[i]).toBe(captured[0]);
    });
    it("should keep path identity stable across re-renders", async () => {
      const [Provider, root] = await newProvider();
      const captured: unknown[] = [];
      const C = ({ trigger }: { trigger: number }) => {
        const [{ path }] = Aether.use({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
        });
        captured.push(path);
        return <span>{trigger}</span>;
      };
      const { rerender } = render(
        <Provider>
          <C trigger={1} />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      rerender(
        <Provider>
          <C trigger={2} />
        </Provider>,
      );
      rerender(
        <Provider>
          <C trigger={3} />
        </Provider>,
      );
      expect(captured.length).toBeGreaterThanOrEqual(3);
      for (let i = 1; i < captured.length; i++) expect(captured[i]).toBe(captured[0]);
    });
    it("should keep methods identity stable when methodsSchema reference is stable", async () => {
      const [Provider, root] = await newProvider();
      const captured: unknown[] = [];
      const C = ({ trigger }: { trigger: number }) => {
        const [, , , methods] = Aether.use({
          type: InvokeLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          methods: invokeMethodsSchema,
        });
        captured.push(methods);
        return <span>{trigger}</span>;
      };
      const { rerender } = render(
        <Provider>
          <C trigger={1} />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      rerender(
        <Provider>
          <C trigger={2} />
        </Provider>,
      );
      rerender(
        <Provider>
          <C trigger={3} />
        </Provider>,
      );
      expect(captured.length).toBeGreaterThanOrEqual(3);
      for (let i = 1; i < captured.length; i++) expect(captured[i]).toBe(captured[0]);
    });
    it("should ignore changes to initialState after the first render", async () => {
      const [Provider, root] = await newProvider();
      const C = ({ x }: { x: number }) => {
        Aether.use({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x },
        });
        return null;
      };
      const { rerender } = render(
        <Provider>
          <C x={1} />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      const leaf = root.children[0] as ExampleLeaf;
      expect(leaf.state.x).toBe(1);
      rerender(
        <Provider>
          <C x={99} />
        </Provider>,
      );
      await new Promise((r) => setTimeout(r, 50));
      expect(leaf.state.x).toBe(1);
    });
    it("should ignore type prop changes after the first render", async () => {
      // The `type` prop is component identity, not a normal prop: it is read
      // once at register time and ignored on subsequent renders. Consumers
      // pass module-level constants here in all real usage; changing it would
      // mean "I want a different worker component", which is what unmount +
      // remount is for.
      const [Provider, root] = await newProvider();
      const C = ({ type }: { type: string }) => {
        Aether.use({
          type,
          schema: exampleProps,
          initialState: { x: 7 },
          aetherKey: "stable-key",
        });
        return null;
      };
      const { rerender } = render(
        <Provider>
          <C type={ExampleLeaf.TYPE} />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      const original = root.children[0] as ExampleLeaf;
      expect(original.type).toBe(ExampleLeaf.TYPE);
      rerender(
        <Provider>
          <C type={InvokeLeaf.TYPE} />
        </Provider>,
      );
      await new Promise((r) => setTimeout(r, 50));
      expect(root.children.length).toBe(1);
      expect(root.children[0]).toBe(original);
      expect(original.deletef).not.toHaveBeenCalled();
      expect(original.type).toBe(ExampleLeaf.TYPE);
    });
    it("should ignore aetherKey changes after the first render", async () => {
      const [Provider, root] = await newProvider();
      const C = ({ k }: { k: string }) => {
        Aether.use({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          aetherKey: k,
        });
        return null;
      };
      const { rerender } = render(
        <Provider>
          <C k="first" />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      const original = root.children[0] as ExampleLeaf;
      expect(original.key).toBe("first");
      rerender(
        <Provider>
          <C k="second" />
        </Provider>,
      );
      await new Promise((r) => setTimeout(r, 50));
      expect(root.children.length).toBe(1);
      expect(root.children[0]).toBe(original);
      expect(original.key).toBe("first");
    });
    it("should throw when initialState fails schema validation", async () => {
      const [Provider] = await newProvider();
      const captured: CapturedError = { current: null };
      const C = () => {
        Aether.use({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          // @ts-expect-error — intentionally invalid for the test
          initialState: { x: "not a number" },
        });
        return null;
      };
      const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      render(
        <ErrorBoundary captured={captured}>
          <Provider>
            <C />
          </Provider>
        </ErrorBoundary>,
      );
      await expect.poll(() => captured.current !== null).toBe(true);
      expect(captured.current?.message).toMatch(/number/i);
      errorSpy.mockRestore();
    });
    it("should throw when setState is called with state that fails schema validation", async () => {
      const [Provider, root] = await newProvider();
      const captured: { error: Error | null } = { error: null };
      const C = () => {
        const [, , setState] = Aether.use({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
        });
        const called = useRef(false);
        if (!called.current) {
          called.current = true;
          try {
            // @ts-expect-error — intentionally invalid for the test
            setState({ x: "not a number" });
          } catch (e) {
            captured.error = e as Error;
          }
        }
        return null;
      };
      render(
        <Provider>
          <C />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      await expect.poll(() => captured.error !== null).toBe(true);
      expect(captured.error?.message).toMatch(/number/i);
    });
    it("should remain functional under StrictMode", async () => {
      const [Provider, root] = await newProvider();
      let observedX: number | null = null;
      const C = () => {
        const [, state, , methods] = Aether.use({
          type: InvokeLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          methods: invokeMethodsSchema,
        });
        observedX = state.x;
        useEffect(() => {
          methods.updateState(55);
        }, [methods]);
        return null;
      };
      render(
        <StrictMode>
          <Provider>
            <C />
          </Provider>
        </StrictMode>,
      );
      await expect.poll(() => observedX === 55).toBe(true);
      await expect.poll(() => root.children.length === 1).toBe(true);
      expect((root.children[0] as InvokeLeaf).state.x).toBe(55);
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
    it("should send the state prop as the initial worker state on first render", async () => {
      const [Provider, root] = await newProvider();
      const C = () => {
        Aether.useUnidirectional({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          state: { x: 123 },
        });
        return null;
      };
      render(
        <Provider>
          <C />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      expect((root.children[0] as ExampleLeaf).state).toEqual({ x: 123 });
    });
    it("should support method invocation through methods", async () => {
      const [Provider, root] = await newProvider();
      const captured: { methods: unknown } = { methods: null };
      const C = () => {
        const { methods } = Aether.useUnidirectional({
          type: InvokeLeaf.TYPE,
          schema: exampleProps,
          state: { x: 0 },
          methods: invokeMethodsSchema,
        });
        captured.methods = methods;
        useEffect(() => {
          methods.fireAndForget();
        }, [methods]);
        return null;
      };
      render(
        <Provider>
          <C />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      const leaf = root.children[0] as InvokeLeaf;
      await expect.poll(() => leaf.fireAndForgetSpy.mock.calls.length > 0).toBe(true);
    });
    it("should return a path that ends with the component's generated key", async () => {
      const [Provider, root] = await newProvider();
      const captured: { path: readonly string[] | null } = { path: null };
      const C = () => {
        const { path } = Aether.useUnidirectional({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          state: { x: 0 },
          aetherKey: "uni-key",
        });
        captured.path = path;
        return null;
      };
      render(
        <Provider>
          <C />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      expect(captured.path).toEqual(["root", "uni-key"]);
      expect((root.children[0] as ExampleLeaf).key).toBe("uni-key");
    });
    it("should propagate rapid state-prop changes to the worker", async () => {
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
          <UnidirectionalLeaf x={0} />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      const leaf = root.children[0] as ExampleLeaf;
      for (let i = 1; i <= 5; i++)
        rerender(
          <Provider>
            <UnidirectionalLeaf x={i} />
          </Provider>,
        );
      await expect.poll(() => leaf.state.x === 5).toBe(true);
    });
    it("should not internally re-render the consumer between prop changes", async () => {
      const [Provider, root] = await newProvider();
      let renderCount = 0;
      const C = ({ x }: { x: number }) => {
        renderCount++;
        Aether.useUnidirectional({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          state: { x },
        });
        return null;
      };
      render(
        <Provider>
          <C x={1} />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      await new Promise((r) => setTimeout(r, 75));
      const baseline = renderCount;
      await new Promise((r) => setTimeout(r, 75));
      expect(renderCount).toBe(baseline);
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
    it("should send the initialState as the worker's first state", async () => {
      const [Provider, root] = await newProvider();
      const C = () => {
        Aether.useLifecycle({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 77 },
        });
        return null;
      };
      render(
        <Provider>
          <C />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      expect((root.children[0] as ExampleLeaf).state).toEqual({ x: 77 });
    });
    it("should expose a setState that updates the worker", async () => {
      const [Provider, root] = await newProvider();
      const captured: { setState: ((s: { x: number }) => void) | null } = {
        setState: null,
      };
      const C = () => {
        const { setState } = Aether.useLifecycle({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
        });
        captured.setState = setState;
        return null;
      };
      render(
        <Provider>
          <C />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      captured.setState?.({ x: 31 });
      await expect
        .poll(() => (root.children[0] as ExampleLeaf).state.x === 31)
        .toBe(true);
    });
    it("should expose methods bound to the worker component", async () => {
      const [Provider, root] = await newProvider();
      const captured: { methods: ReturnType<typeof Aether.useLifecycle> | null } = {
        methods: null,
      };
      const C = () => {
        const handle = Aether.useLifecycle({
          type: InvokeLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          methods: invokeMethodsSchema,
        });
        captured.methods = handle;
        useEffect(() => {
          handle.methods.updateState(82);
        }, [handle.methods]);
        return null;
      };
      render(
        <Provider>
          <C />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      await expect
        .poll(() => (root.children[0] as InvokeLeaf).state.x === 82)
        .toBe(true);
    });
    it("should return a path of [...parentPath, generatedKey]", async () => {
      const [Provider, root] = await newProvider();
      const captured: { path: readonly string[] | null } = { path: null };
      const C = () => {
        const { path } = Aether.useLifecycle({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          aetherKey: "lifecycle-key",
        });
        captured.path = path;
        return null;
      };
      render(
        <Provider>
          <C />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      expect(captured.path).toEqual(["root", "lifecycle-key"]);
    });
    it("should not re-render the consumer on worker state pushes", async () => {
      const [Provider, root] = await newProvider();
      let renderCount = 0;
      const C = () => {
        renderCount++;
        const { methods } = Aether.useLifecycle({
          type: InvokeLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          methods: invokeMethodsSchema,
        });
        useEffect(() => {
          methods.updateState(11);
          methods.updateState(22);
          methods.updateState(33);
        }, [methods]);
        return null;
      };
      render(
        <Provider>
          <C />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      await expect
        .poll(() => (root.children[0] as InvokeLeaf).state.x === 33)
        .toBe(true);
      const settled = renderCount;
      await new Promise((r) => setTimeout(r, 75));
      expect(renderCount).toBe(settled);
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
    it("should give two siblings under one Composite distinct paths", async () => {
      const [Provider, root] = await newProvider();
      const Parent = () => {
        const [{ path }] = Aether.use({
          type: ExampleComposite.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
        });
        return (
          <Aether.Composite path={path}>
            <Child aetherKey="a" x={1} />
            <Child aetherKey="b" x={2} />
          </Aether.Composite>
        );
      };
      const Child = ({ aetherKey, x }: { aetherKey: string; x: number }) => {
        Aether.use({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x },
          aetherKey,
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
      await expect.poll(() => composite.children.length === 2).toBe(true);
      const keys = composite.children.map((c) => c.key).sort();
      expect(keys).toEqual(["a", "b"]);
      const byKey = Object.fromEntries(
        composite.children.map((c) => [c.key, c]),
      ) as Record<string, ExampleLeaf>;
      expect(byKey.a.state.x).toBe(1);
      expect(byKey.b.state.x).toBe(2);
    });
    it("should support arbitrarily deep Composite > Composite > Leaf nesting", async () => {
      const [Provider, root] = await newProvider();
      const Outer = () => {
        const [{ path }] = Aether.use({
          type: ExampleComposite.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          aetherKey: "outer",
        });
        return (
          <Aether.Composite path={path}>
            <Inner />
          </Aether.Composite>
        );
      };
      const Inner = () => {
        const [{ path }] = Aether.use({
          type: ExampleComposite.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          aetherKey: "inner",
        });
        return (
          <Aether.Composite path={path}>
            <Leaf />
          </Aether.Composite>
        );
      };
      const Leaf = () => {
        Aether.use({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 9 },
          aetherKey: "leaf",
        });
        return null;
      };
      render(
        <Provider>
          <Outer />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      const outer = root.children[0] as ExampleComposite;
      expect(outer.key).toBe("outer");
      await expect.poll(() => outer.children.length === 1).toBe(true);
      const inner = outer.children[0] as ExampleComposite;
      expect(inner.key).toBe("inner");
      await expect.poll(() => inner.children.length === 1).toBe(true);
      const leaf = inner.children[0] as ExampleLeaf;
      expect(leaf.key).toBe("leaf");
      expect(leaf.state.x).toBe(9);
    });
    it("should preserve child worker components across Composite re-renders", async () => {
      const [Provider, root] = await newProvider();
      const Parent = ({ trigger }: { trigger: number }) => {
        const [{ path }] = Aether.use({
          type: ExampleComposite.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          aetherKey: "parent",
        });
        return (
          <Aether.Composite path={path}>
            <Child trigger={trigger} />
          </Aether.Composite>
        );
      };
      const Child = ({ trigger }: { trigger: number }) => {
        Aether.use({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          aetherKey: "child",
        });
        return <span>{trigger}</span>;
      };
      const { rerender } = render(
        <Provider>
          <Parent trigger={1} />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      const composite = root.children[0] as ExampleComposite;
      await expect.poll(() => composite.children.length === 1).toBe(true);
      const originalLeaf = composite.children[0] as ExampleLeaf;
      rerender(
        <Provider>
          <Parent trigger={2} />
        </Provider>,
      );
      rerender(
        <Provider>
          <Parent trigger={3} />
        </Provider>,
      );
      await new Promise((r) => setTimeout(r, 50));
      expect(composite.children.length).toBe(1);
      expect(composite.children[0]).toBe(originalLeaf);
      expect(originalLeaf.deletef).not.toHaveBeenCalled();
    });
    it("should keep a child's path reference stable across Composite re-renders", async () => {
      const [Provider, root] = await newProvider();
      const captured: unknown[] = [];
      const Parent = ({ trigger }: { trigger: number }) => {
        const [{ path }] = Aether.use({
          type: ExampleComposite.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          aetherKey: "parent-stable",
        });
        return (
          <Aether.Composite path={path}>
            <Child trigger={trigger} />
          </Aether.Composite>
        );
      };
      const Child = ({ trigger }: { trigger: number }) => {
        const [{ path }] = Aether.use({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          aetherKey: "child-stable",
        });
        captured.push(path);
        return <span>{trigger}</span>;
      };
      const { rerender } = render(
        <Provider>
          <Parent trigger={1} />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      rerender(
        <Provider>
          <Parent trigger={2} />
        </Provider>,
      );
      rerender(
        <Provider>
          <Parent trigger={3} />
        </Provider>,
      );
      expect(captured.length).toBeGreaterThanOrEqual(3);
      for (let i = 1; i < captured.length; i++) expect(captured[i]).toBe(captured[0]);
    });
  });
  describe("Provider", () => {
    it("should surface worker-reported errors through the render tree", async () => {
      const [Provider] = await newProvider();
      const captured: CapturedError = { current: null };
      const C = () => {
        Aether.use({
          type: ErrorOnUpdateLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
        });
        return null;
      };
      const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      render(
        <ErrorBoundary captured={captured}>
          <Provider>
            <C />
          </Provider>
        </ErrorBoundary>,
      );
      await expect.poll(() => captured.current !== null).toBe(true);
      expect(captured.current?.message).toMatch(
        /ErrorOnUpdateLeaf afterUpdate failure/,
      );
      errorSpy.mockRestore();
    });
    it("should render multiple independent children under one Provider", async () => {
      const [Provider, root] = await newProvider();
      const A = () => {
        Aether.use({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 1 },
          aetherKey: "a",
        });
        return null;
      };
      const B = () => {
        Aether.use({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 2 },
          aetherKey: "b",
        });
        return null;
      };
      render(
        <Provider>
          <A />
          <B />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 2).toBe(true);
      const byKey = Object.fromEntries(root.children.map((c) => [c.key, c])) as Record<
        string,
        ExampleLeaf
      >;
      expect(byKey.a.state.x).toBe(1);
      expect(byKey.b.state.x).toBe(2);
    });
    it("should clean up all child worker components on Provider unmount", async () => {
      const [Provider, root] = await newProvider();
      const A = () => {
        Aether.use({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          aetherKey: "cleanup-a",
        });
        return null;
      };
      const B = () => {
        Aether.use({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          aetherKey: "cleanup-b",
        });
        return null;
      };
      const { unmount } = render(
        <Provider>
          <A />
          <B />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 2).toBe(true);
      const leafA = root.children.find((c) => c.key === "cleanup-a") as ExampleLeaf;
      const leafB = root.children.find((c) => c.key === "cleanup-b") as ExampleLeaf;
      unmount();
      await expect.poll(() => leafA.deletef.mock.calls.length > 0).toBe(true);
      await expect.poll(() => leafB.deletef.mock.calls.length > 0).toBe(true);
    });
    it("should no-op when workerEnabled is false", async () => {
      const captured: CapturedError = { current: null };
      let renderedFine = false;
      const C = () => {
        const [, , setState] = Aether.use({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
        });
        const did = useRef(false);
        if (!did.current) {
          did.current = true;
          setState({ x: 1 });
        }
        renderedFine = true;
        return null;
      };
      render(
        <ErrorBoundary captured={captured}>
          <Aether.Provider workerEnabled={false}>
            <C />
          </Aether.Provider>
        </ErrorBoundary>,
      );
      await expect.poll(() => renderedFine).toBe(true);
      expect(captured.current).toBeNull();
    });
    it("should throw when constructed without a worker source", () => {
      expect(() => new Aether.Store()).toThrow(/neither `worker` nor `workerURL`/);
    });
  });
  describe("worker transport errors", () => {
    let instances: FakeWorker[];
    beforeEach(() => {
      instances = [];
      class TrackedWorker extends FakeWorker {
        constructor(url: string | URL) {
          super(url);
          instances.push(this);
        }
      }
      vi.stubGlobal("Worker", TrackedWorker);
    });
    afterEach(() => {
      vi.unstubAllGlobals();
    });
    it("should surface worker.onerror events with location info", () => {
      const store = new Aether.Store({ workerURL: "test://worker" });
      const listener = vi.fn();
      store.subscribeError(listener);
      expect(store.getError()).toBeNull();
      instances[0].onerror?.(
        new ErrorEvent("error", {
          message: "boom",
          filename: "worker.js",
          lineno: 42,
          colno: 7,
        }),
      );
      expect(listener).toHaveBeenCalledTimes(1);
      expect(store.getError()?.message).toBe(
        "[aether] worker error: boom (worker.js:42:7)",
      );
      store.dispose();
    });
    it("should fall back to 'unknown' when ErrorEvent carries no message", () => {
      const store = new Aether.Store({ workerURL: "test://worker" });
      instances[0].onerror?.(new ErrorEvent("error"));
      expect(store.getError()?.message).toBe("[aether] worker error: unknown");
      store.dispose();
    });
    it("should surface worker.onmessageerror as a deserialization failure", () => {
      const store = new Aether.Store({ workerURL: "test://worker" });
      const listener = vi.fn();
      store.subscribeError(listener);
      instances[0].onmessageerror?.(new MessageEvent("messageerror"));
      expect(listener).toHaveBeenCalledTimes(1);
      expect(store.getError()?.message).toBe(
        "[aether] failed to deserialize message from worker",
      );
      store.dispose();
    });
  });
  describe("regression pins", () => {
    it("should silently collapse two siblings sharing an aetherKey to one worker entry (current behavior)", async () => {
      // Pin: today two components rendered under the same Provider with the same
      // aetherKey end up sharing a single worker entry; the second registration
      // overwrites the first's handler. The refactor may make this an error.
      const [Provider, root] = await newProvider();
      const C = ({ x }: { x: number }) => {
        Aether.use({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x },
          aetherKey: "duplicate",
        });
        return null;
      };
      render(
        <Provider>
          <C x={1} />
          <C x={2} />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      const leaf = root.children[0] as ExampleLeaf;
      expect(leaf.key).toBe("duplicate");
      expect(leaf.state.x).toBe(2);
    });
    it("should keep leaves that share a key under different parents independent", async () => {
      // Two plots charting the same channel produce lines with identical keys. Identity
      // is the full path, not the leaf key, so neither plot evicts the other's line.
      const [Provider, root] = await newProvider();
      const Plot = ({ aetherKey, x }: { aetherKey: string; x: number }) => {
        const [{ path }] = Aether.use({
          type: ExampleComposite.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
          aetherKey,
        });
        return (
          <Aether.Composite path={path}>
            <Line x={x} />
          </Aether.Composite>
        );
      };
      const Line = ({ x }: { x: number }) => {
        Aether.use({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x },
          aetherKey: "shared-line",
        });
        return null;
      };
      render(
        <Provider>
          <Plot aetherKey="plot-a" x={1} />
          <Plot aetherKey="plot-b" x={2} />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 2).toBe(true);
      const byKey = Object.fromEntries(
        root.children.map((c) => [c.key, c as ExampleComposite]),
      );
      const plotA = byKey["plot-a"];
      const plotB = byKey["plot-b"];
      await expect.poll(() => plotA.children.length === 1).toBe(true);
      await expect.poll(() => plotB.children.length === 1).toBe(true);
      const lineA = plotA.children[0] as ExampleLeaf;
      const lineB = plotB.children[0] as ExampleLeaf;
      expect(lineA.key).toBe("shared-line");
      expect(lineB.key).toBe("shared-line");
      expect(lineA).not.toBe(lineB);
      expect(lineA.state.x).toBe(1);
      expect(lineB.state.x).toBe(2);
      lineB.setState({ x: 99 });
      expect(lineA.state.x).toBe(1);
      expect(lineB.state.x).toBe(99);
    });
    it("should create a fresh worker component when an aetherKey is reused after unmount", async () => {
      const [Provider, root] = await newProvider();
      const C = () => {
        Aether.use({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 5 },
          aetherKey: "reused",
        });
        return null;
      };
      const first = render(
        <Provider>
          <C />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      const original = root.children[0] as ExampleLeaf;
      first.unmount();
      await expect.poll(() => original.deletef.mock.calls.length > 0).toBe(true);
      render(
        <Provider>
          <C />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      const fresh = root.children[0] as ExampleLeaf;
      expect(fresh).not.toBe(original);
      expect(fresh.state.x).toBe(5);
    });
    it("should apply worker state pushes that arrive during the mount window", async () => {
      // Pin: today, when the worker pushes state asynchronously in response to the
      // synchronous initial-state send, the consumer's rendered state eventually
      // reflects the push. The refactor should preserve this end-state behavior while
      // eliminating the React warnings that currently fire along the way (see the
      // warning pin below).
      const [Provider, root] = await newProvider();
      let observedX: number | null = null;
      const C = () => {
        const [, state] = Aether.use({
          type: EchoOnUpdateLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
        });
        observedX = state.x;
        return null;
      };
      render(
        <Provider>
          <C />
        </Provider>,
      );
      await waitFor(() => {
        expect(root.children.length).toBe(1);
        expect(observedX).toBe(1);
      });
      expect((root.children[0] as EchoOnUpdateLeaf).state.x).toBe(1);
    });
    it("should not fire the 'hasn't mounted yet' warning when worker pushes state during the mount window", async () => {
      const [Provider, root] = await newProvider();
      const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      const C = () => {
        Aether.use({
          type: EchoOnUpdateLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
        });
        return null;
      };
      render(
        <Provider>
          <C />
        </Provider>,
      );
      await expect.poll(() => root.children.length === 1).toBe(true);
      const leaf = root.children[0] as EchoOnUpdateLeaf;
      await expect.poll(() => leaf.state.x === 1).toBe(true);
      const hasMountWarning = errorSpy.mock.calls.some(([msg]) => {
        const s =
          typeof msg === "string"
            ? msg
            : msg instanceof Error
              ? msg.message
              : String(msg);
        return /hasn't mounted yet/.test(s);
      });
      expect(hasMountWarning).toBe(false);
      errorSpy.mockRestore();
    });
    it("should return a cached snapshot while a subscriber outlives the entry", () => {
      // Pin: useSyncExternalStore may call getSnapshot for tearing detection after the
      // entry has been unregistered (e.g. StrictMode's pseudo- unmount/remount window).
      // The store should fall back to the last known snapshot for that path rather than
      // throw.
      const [workerSide, mainSide] = aether.createMockPair();
      aether.render({ worker: workerSide, registry: REGISTRY });
      const store = new Aether.Store({ worker: mainSide });
      const path = ["root", "pinned"];
      const snapshot = () => store.getSnapshot<typeof exampleProps>(path);
      const unsubscribe = store.subscribe(path, () => {});
      store.register({
        type: ExampleLeaf.TYPE,
        path,
        schema: exampleProps,
        initialState: { x: 7 },
      });
      expect(snapshot()).toEqual({ x: 7 });
      store.unregister(path);
      // Subscriber still attached: cached snapshot remains readable.
      expect(snapshot()).toEqual({ x: 7 });
      unsubscribe();
      // No subscribers and no entry: cache is cleared.
      expect(snapshot).toThrow(/missing entry/);
    });
  });
});
