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
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Handle } from "@/schematic/node/common/handle";

const mocks = vi.hoisted(() => ({
  updateInternals: vi.fn(),
  shouldThrow: false,
}));

vi.mock("@xyflow/react", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    useUpdateNodeInternals: () => {
      if (mocks.shouldThrow) throw new Error("not in RF context");
      return mocks.updateInternals;
    },
  };
});

const RFNode = ({
  id,
  children,
}: {
  id?: string;
  children: ReactElement;
}): ReactElement => (
  <div className="react-flow__node" data-id={id}>
    {children}
  </div>
);

describe("Handle.Boundary", () => {
  beforeEach(() => {
    mocks.updateInternals.mockReset();
    mocks.shouldThrow = false;
  });

  afterEach(() => {
    mocks.updateInternals.mockReset();
  });

  describe("fallback when xyflow is unavailable", () => {
    it("should render null when useUpdateNodeInternals throws", () => {
      mocks.shouldThrow = true;
      const { container } = render(
        <Handle.Boundary orientation="left">
          <span data-testid="child">child</span>
        </Handle.Boundary>,
      );
      expect(container.querySelector('[data-testid="child"]')).toBeNull();
      expect(container.firstChild).toBeNull();
    });

    it("should not call updateInternals when the hook throws", () => {
      mocks.shouldThrow = true;
      render(
        <Handle.Boundary orientation="left">
          <span>child</span>
        </Handle.Boundary>,
      );
      expect(mocks.updateInternals).not.toHaveBeenCalled();
    });
  });

  describe("first-render skip", () => {
    it("should NOT call updateInternals on initial mount", () => {
      render(
        <RFNode id="n1">
          <Handle.Boundary orientation="left">
            <span data-testid="child">child</span>
          </Handle.Boundary>
        </RFNode>,
      );
      expect(mocks.updateInternals).not.toHaveBeenCalled();
    });

    it("should render its children on initial mount", () => {
      const { container } = render(
        <RFNode id="n1">
          <Handle.Boundary orientation="left">
            <span data-testid="child">visible</span>
          </Handle.Boundary>
        </RFNode>,
      );
      expect(container.querySelector('[data-testid="child"]')).not.toBeNull();
    });
  });

  describe("update triggers on dependency change", () => {
    it("should call updateInternals with the parent node id when orientation changes", () => {
      const { rerender } = render(
        <RFNode id="my-node">
          <Handle.Boundary orientation="left">
            <span>child</span>
          </Handle.Boundary>
        </RFNode>,
      );
      expect(mocks.updateInternals).not.toHaveBeenCalled();
      rerender(
        <RFNode id="my-node">
          <Handle.Boundary orientation="top">
            <span>child</span>
          </Handle.Boundary>
        </RFNode>,
      );
      expect(mocks.updateInternals).toHaveBeenCalledTimes(1);
      expect(mocks.updateInternals).toHaveBeenCalledWith("my-node");
    });

    it("should call updateInternals when refreshDeps changes", () => {
      const { rerender } = render(
        <RFNode id="my-node">
          <Handle.Boundary orientation="left" refreshDeps={1}>
            <span>child</span>
          </Handle.Boundary>
        </RFNode>,
      );
      rerender(
        <RFNode id="my-node">
          <Handle.Boundary orientation="left" refreshDeps={2}>
            <span>child</span>
          </Handle.Boundary>
        </RFNode>,
      );
      expect(mocks.updateInternals).toHaveBeenCalledTimes(1);
      expect(mocks.updateInternals).toHaveBeenCalledWith("my-node");
    });

    it("should not re-fire updateInternals when neither orientation nor refreshDeps changes", () => {
      const { rerender } = render(
        <RFNode id="my-node">
          <Handle.Boundary orientation="left" refreshDeps="x">
            <span>child</span>
          </Handle.Boundary>
        </RFNode>,
      );
      rerender(
        <RFNode id="my-node">
          <Handle.Boundary orientation="left" refreshDeps="x">
            <span>child</span>
          </Handle.Boundary>
        </RFNode>,
      );
      expect(mocks.updateInternals).not.toHaveBeenCalled();
    });
  });

  describe("data-id resolution", () => {
    it("should bail out silently when no react-flow__node ancestor exists", () => {
      const { rerender } = render(
        <Handle.Boundary orientation="left">
          <span>child</span>
        </Handle.Boundary>,
      );
      rerender(
        <Handle.Boundary orientation="top">
          <span>child</span>
        </Handle.Boundary>,
      );
      expect(mocks.updateInternals).not.toHaveBeenCalled();
    });

    it("should bail out when the ancestor node is missing its data-id", () => {
      const { rerender } = render(
        <div className="react-flow__node">
          <Handle.Boundary orientation="left">
            <span>child</span>
          </Handle.Boundary>
        </div>,
      );
      rerender(
        <div className="react-flow__node">
          <Handle.Boundary orientation="top">
            <span>child</span>
          </Handle.Boundary>
        </div>,
      );
      expect(mocks.updateInternals).not.toHaveBeenCalled();
    });
  });
});
