// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { type ReactElement } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Handle } from "@/schematic/node/common/handle";

const mocks = vi.hoisted(() => ({
  updateInternals: vi.fn(),
  nodeID: null as string | null,
}));

vi.mock("@xyflow/react", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    useNodeId: () => mocks.nodeID,
    useUpdateNodeInternals: () => {
      if (mocks.nodeID == null) throw new Error("not in RF context");
      return mocks.updateInternals;
    },
  };
});

const boundary = (props: Partial<Handle.BoundaryProps> = {}): ReactElement => (
  <Handle.Boundary orientation="left" {...props}>
    <span data-testid="child">child</span>
  </Handle.Boundary>
);

describe("Handle.Boundary", () => {
  beforeEach(() => {
    mocks.updateInternals.mockReset();
    mocks.nodeID = null;
  });

  describe("outside a React Flow node", () => {
    it("should render null", () => {
      const { container } = render(boundary());
      expect(container.firstChild).toBeNull();
    });

    it("should not call updateInternals when the orientation changes", () => {
      const { rerender } = render(boundary());
      rerender(boundary({ orientation: "top" }));
      expect(mocks.updateInternals).not.toHaveBeenCalled();
    });
  });

  describe("first-render skip", () => {
    it("should NOT call updateInternals on initial mount", () => {
      mocks.nodeID = "n1";
      render(boundary());
      expect(mocks.updateInternals).not.toHaveBeenCalled();
    });

    it("should render its children on initial mount", () => {
      mocks.nodeID = "n1";
      const { getByTestId } = render(boundary());
      expect(getByTestId("child").textContent).toBe("child");
    });
  });

  describe("update triggers on dependency change", () => {
    it("should call updateInternals with the node id when orientation changes", () => {
      mocks.nodeID = "my-node";
      const { rerender } = render(boundary());
      expect(mocks.updateInternals).not.toHaveBeenCalled();
      rerender(boundary({ orientation: "top" }));
      expect(mocks.updateInternals).toHaveBeenCalledTimes(1);
      expect(mocks.updateInternals).toHaveBeenCalledWith("my-node");
    });

    it("should call updateInternals when refreshDeps changes", () => {
      mocks.nodeID = "my-node";
      const { rerender } = render(boundary({ refreshDeps: 1 }));
      rerender(boundary({ refreshDeps: 2 }));
      expect(mocks.updateInternals).toHaveBeenCalledTimes(1);
      expect(mocks.updateInternals).toHaveBeenCalledWith("my-node");
    });

    it("should not re-fire updateInternals when neither orientation nor refreshDeps changes", () => {
      mocks.nodeID = "my-node";
      const { rerender } = render(boundary({ refreshDeps: "x" }));
      rerender(boundary({ refreshDeps: "x" }));
      expect(mocks.updateInternals).not.toHaveBeenCalled();
    });
  });
});
