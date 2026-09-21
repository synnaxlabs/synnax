// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, render } from "@testing-library/react";
import {
  ReactFlow,
  type ReactFlowInstance,
  ReactFlowProvider,
  useReactFlow,
  useStoreApi,
  type Viewport,
} from "@xyflow/react";
import { type ReactElement } from "react";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";

import { useOnViewportChange } from "@/vis/diagram/useOnViewportChange";

const MOUNTED: Viewport = { x: 10, y: 20, zoom: 0.75 };

interface Api {
  flow: ReactFlowInstance;
  store: ReturnType<typeof useStoreApi>;
}

interface ProbeProps {
  onChange: (viewport: Viewport) => void;
  api: { current: Api | null };
}

const Probe = ({ onChange, api }: ProbeProps): null => {
  useOnViewportChange(onChange);
  api.current = { flow: useReactFlow(), store: useStoreApi() };
  return null;
};

interface TreeProps extends ProbeProps {
  mounted: boolean;
}

// React Flow mounts as a sibling of the probe, the way the diagram conditionally
// mounts it under one long-lived provider.
const Tree = ({ mounted, ...probe }: TreeProps): ReactElement => (
  <ReactFlowProvider>
    {mounted && <ReactFlow defaultViewport={MOUNTED} minZoom={0.5} maxZoom={1} />}
    <Probe {...probe} />
  </ReactFlowProvider>
);

const renderTree = () => {
  const onChange = vi.fn<(viewport: Viewport) => void>();
  const api: { current: Api | null } = { current: null };
  const { rerender } = render(<Tree mounted onChange={onChange} api={api} />);
  return {
    onChange,
    api: () => {
      if (api.current == null) throw new Error("probe not rendered");
      return api.current;
    },
    mount: (mounted: boolean) =>
      rerender(<Tree mounted={mounted} onChange={onChange} api={api} />),
  };
};

describe("Diagram.useOnViewportChange", () => {
  // jsdom lays nothing out; React Flow sizes its container from these.
  beforeAll(() => {
    vi.spyOn(HTMLElement.prototype, "offsetWidth", "get").mockReturnValue(1000);
    vi.spyOn(HTMLElement.prototype, "offsetHeight", "get").mockReturnValue(500);
  });
  afterAll(() => vi.restoreAllMocks());

  it("reports the viewport React Flow mounts at", () => {
    const { onChange } = renderTree();
    expect(onChange).toHaveBeenLastCalledWith(MOUNTED);
  });

  it("reports a viewport set through React Flow", () => {
    const { onChange, api } = renderTree();
    const next = { x: 1, y: 2, zoom: 1 };
    act(() => void api().flow.setViewport(next));
    expect(onChange).toHaveBeenLastCalledWith(next);
  });

  it("reports a transform written straight to the store", () => {
    const { onChange, api } = renderTree();
    act(() => api().store.setState({ transform: [5, 6, 0.5] }));
    expect(onChange).toHaveBeenLastCalledWith({ x: 5, y: 6, zoom: 0.5 });
  });

  it("ignores store changes that leave the transform alone", () => {
    const { onChange, api } = renderTree();
    onChange.mockClear();
    act(() => api().store.setState({ nodesSelectionActive: true }));
    expect(onChange).not.toHaveBeenCalled();
  });

  // React Flow resets its store when it unmounts, which wipes the callbacks React
  // Flow's own useOnViewportChange registers once.
  it("keeps reporting after React Flow unmounts and remounts", () => {
    const { onChange, api, mount } = renderTree();
    mount(false);
    onChange.mockClear();
    mount(true);
    expect(onChange).toHaveBeenLastCalledWith(MOUNTED);
    const next = { x: 1, y: 2, zoom: 1 };
    act(() => void api().flow.setViewport(next));
    expect(onChange).toHaveBeenLastCalledWith(next);
  });

  it("does not report the placeholder transform of an unmounted React Flow", () => {
    const { onChange, api, mount } = renderTree();
    onChange.mockClear();
    mount(false);
    expect(api().store.getState().transform).toEqual([0, 0, 1]);
    expect(onChange).not.toHaveBeenCalled();
  });
});
