// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, createTestClient } from "@synnaxlabs/client";
import { id, xy } from "@synnaxlabs/x";
import { act, render, renderHook, waitFor, within } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { Arc } from "@/arc";
import { useClipboard } from "@/arc/graph/clipboard";
import { Errors } from "@/errors";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { type Diagram } from "@/vis/diagram";

const client = createTestClient();
let wrapper: FC<PropsWithChildren>;

beforeAll(async () => {
  wrapper = await createAsyncSynnaxWrapper({ client });
});

const N1 = "n1";
const N2 = "n2";
const EDGE_KEY = arc.ir.edgeKey({ node: N1, param: "out" }, { node: N2, param: "in" });

const createGraphArc = async (): Promise<arc.Arc> =>
  await client.arcs.create({
    name: `clipboard_${id.create()}`,
    mode: "graph",
    graph: {
      nodes: [
        { key: N1, position: { x: 0, y: 0 } },
        { key: N2, position: { x: 10, y: 10 } },
      ],
      edges: [
        {
          source: { node: N1, param: "out" },
          target: { node: N2, param: "in" },
          kind: arc.ir.EdgeKind.continuous,
        },
      ],
      configs: { [N1]: { type: "constant" }, [N2]: { type: "log" } },
      functions: [],
    },
    text: { raw: "" },
  });

// loadArc primes the flux store with the arc at key via the suspending
// useEnsureRetrieved. A single-hook bootstrap keeps the suspending hook from
// being followed by additional hooks, which trips a React 19 replay warning.
const loadArc = async (key: string): Promise<void> => {
  const Wrapper = wrapper;
  const Bootstrap = (): ReactElement => {
    Arc.useEnsureRetrieved({ key });
    return <div data-testid="loaded" />;
  };
  let utils!: ReturnType<typeof render>;
  await act(async () => {
    utils = render(
      <Wrapper>
        <Errors.SuspenseBoundary loading={null}>
          <Bootstrap />
        </Errors.SuspenseBoundary>
      </Wrapper>,
    );
  });
  await within(utils.container).findByTestId("loaded");
};

/**
 * fakeClipboardEvent backs the browser clipboard API with an in-memory store so
 * a single event can carry a payload from copy through to paste. jsdom does not
 * implement a functional DataTransfer, so the test supplies one.
 */
const fakeClipboardEvent = () => {
  const store = new Map<string, string>();
  return {
    preventDefault: vi.fn(),
    clipboardData: {
      getData: (mime: string) => store.get(mime) ?? "",
      setData: (mime: string, data: string) => {
        store.set(mime, data);
      },
    },
  } as unknown as Parameters<Diagram.UseClipboardReturn["onCopy"]>[0];
};

describe("arc graph clipboard", () => {
  const setup = async (selected: string[], onPaste?: (keys: string[]) => void) => {
    const { key } = await createGraphArc();
    await loadArc(key);
    const { result } = renderHook(
      () => ({
        nodes: Arc.useSelectNodes({ key }),
        edges: Arc.useSelectEdges({ key }),
        clipboard: useClipboard({ key, selected, onPaste }),
      }),
      { wrapper },
    );
    return { key, result };
  };

  it("should copy a selected subgraph and paste it into the store as a fresh subgraph", async () => {
    const { result } = await setup([N1, N2, EDGE_KEY]);
    expect(result.current.nodes).toHaveLength(2);

    const e = fakeClipboardEvent();
    await act(async () => {
      result.current.clipboard.onCopy(e, xy.ZERO);
      result.current.clipboard.onPaste(e, xy.construct(105, 205));
    });

    await waitFor(() => expect(result.current.nodes).toHaveLength(4));
    await waitFor(() => expect(result.current.edges).toHaveLength(2));

    const newNodes = result.current.nodes.filter((n) => n.key !== N1 && n.key !== N2);
    expect(newNodes).toHaveLength(2);
    // centroid (5, 5) offset to cursor (105, 205) => (100, 200).
    expect(newNodes.map((n) => n.position)).toEqual([
      { x: 100, y: 200 },
      { x: 110, y: 210 },
    ]);
  });

  it("should drop edges whose endpoints are not part of the copied selection", async () => {
    const { result } = await setup([N1]);

    const e = fakeClipboardEvent();
    await act(async () => {
      result.current.clipboard.onCopy(e, xy.ZERO);
      result.current.clipboard.onPaste(e, xy.ZERO);
    });

    await waitFor(() => expect(result.current.nodes).toHaveLength(3));
    expect(result.current.edges).toHaveLength(1);
  });

  it("should not change the store when the selection copies nothing", async () => {
    const { result } = await setup(["does-not-exist"]);

    const e = fakeClipboardEvent();
    await act(async () => {
      result.current.clipboard.onCopy(e, xy.ZERO);
      result.current.clipboard.onPaste(e, xy.ZERO);
    });

    expect(result.current.nodes).toHaveLength(2);
    expect(result.current.edges).toHaveLength(1);
  });

  it("should invoke onPaste with the freshly pasted node keys", async () => {
    const onPaste = vi.fn();
    const { result } = await setup([N1, N2], onPaste);

    const e = fakeClipboardEvent();
    await act(async () => {
      result.current.clipboard.onCopy(e, xy.ZERO);
      result.current.clipboard.onPaste(e, xy.ZERO);
    });

    await waitFor(() => expect(result.current.nodes).toHaveLength(4));
    const newKeys = result.current.nodes
      .filter((n) => n.key !== N1 && n.key !== N2)
      .map((n) => n.key);
    expect(onPaste).toHaveBeenCalledWith(newKeys);
  });
});
