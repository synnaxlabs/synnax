// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, render } from "@testing-library/react";
import { type Node, ReactFlowProvider, useStoreApi } from "@xyflow/react";
import { type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Jumps } from "@/schematic/edge/common/jumps";
import { Path } from "@/schematic/edge/common/path";
import { Segmented } from "@/schematic/edge/common/segmented";
import { diagram } from "@/vis/diagram/aether";

type Store = ReturnType<typeof useStoreApi>;

const NODES: Node[] = [
  { id: "a", position: { x: 0, y: 0 }, data: {} },
  { id: "b", position: { x: 200, y: 100 }, data: {} },
];

const Line = ({ points, crossings }: Segmented.PathProps): ReactElement => (
  <path d={Path.rounded(points, crossings)} />
);

const spec = Segmented.createSpec("electric", "Electric", Line);

const StoreProbe = ({ onStore }: { onStore: (s: Store) => void }): null => {
  onStore(useStoreApi());
  return null;
};

const renderEdge = () => {
  let store!: Store;
  const jumps = Jumps.createStore();
  const ui = (edgeKey: string): ReactElement => (
    <ReactFlowProvider defaultNodes={NODES}>
      <StoreProbe onStore={(s) => (store = s)} />
      <Jumps.Context value={jumps}>
        <svg>
          <spec.Edge
            key={edgeKey}
            edgeKey={edgeKey}
            source={diagram.createEndpoint(0, 0, "right")}
            target={diagram.createEndpoint(200, 100, "left")}
            sourceNode="a"
            targetNode="b"
            selected={false}
            config={spec.defaultConfig()}
            onChange={vi.fn()}
          />
        </svg>
      </Jumps.Context>
    </ReactFlowProvider>
  );
  const { container, rerender } = render(ui("first"));
  const path = (): string | null =>
    container.querySelector("path")?.getAttribute("d") ?? null;
  return { store: () => store, path, remount: () => rerender(ui("second")) };
};

describe("Segmented", () => {
  // The edge measures its nodes during render, so they must already be in the DOM.
  const nodeElements: HTMLElement[] = [];
  beforeEach(() =>
    NODES.forEach(({ id }) => {
      const el = document.createElement("div");
      el.dataset.id = id;
      document.body.appendChild(el);
      nodeElements.push(el);
    }),
  );
  afterEach(() => nodeElements.splice(0).forEach((el) => el.remove()));

  it("should draw the edge for the current zoom after the viewport zooms", () => {
    const { store, path, remount } = renderEdge();
    const atStart = path();
    act(() => store().setState({ transform: [0, 0, 2] }));
    const zoomed = path();
    remount();
    expect(zoomed).toEqual(path());
    expect(zoomed).not.toEqual(atStart);
  });
});
