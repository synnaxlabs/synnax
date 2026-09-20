// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { type ReactElement, useRef } from "react";
import { createPortal } from "react-dom";
import { describe, expect, it, vi } from "vitest";

import {
  PORTAL_ID_ATTR,
  PORTAL_OWNER_ATTR,
  useClickOutside,
} from "@/dialog/useClickOutside";
import { firePointerDown, mockBoundingClientRect } from "@/testutil/dom";

interface OwnerProps {
  onClickOutside: () => void;
}

const Owner = ({ onClickOutside }: OwnerProps): ReactElement => {
  const ref = useRef<HTMLDivElement>(null);
  useClickOutside({ ref, onClickOutside });
  return (
    <div ref={ref} {...{ [PORTAL_ID_ATTR]: "owner" }}>
      <span>Inside</span>
      {createPortal(
        <div {...{ [PORTAL_OWNER_ATTR]: "owner" }}>
          <span>Portaled</span>
        </div>,
        document.body,
      )}
    </div>
  );
};
Owner.displayName = "Owner";

// jsdom gives every element a zero rect, which reads as containing every click, so
// the owner and the viewport need rects of their own.
const renderOwner = (onClickOutside: () => void) => {
  const c = render(
    <>
      <Owner onClickOutside={onClickOutside} />
      <span>Elsewhere</span>
    </>,
  );
  const owner = c.getByText("Inside").parentElement;
  if (owner == null) throw new Error("owner not found");
  owner.getBoundingClientRect = mockBoundingClientRect(0, 0, 50, 50);
  document.documentElement.getBoundingClientRect = mockBoundingClientRect(
    0,
    0,
    1000,
    1000,
  );
  return c;
};

describe("useClickOutside", () => {
  it("should call the callback when the click lands outside the element", () => {
    const onClickOutside = vi.fn();
    const c = renderOwner(onClickOutside);
    firePointerDown(c.getByText("Elsewhere"), { x: 200, y: 200 });
    expect(onClickOutside).toHaveBeenCalledOnce();
  });

  it("should not call the callback when the click lands inside the element", () => {
    const onClickOutside = vi.fn();
    const c = renderOwner(onClickOutside);
    firePointerDown(c.getByText("Inside"), { x: 10, y: 10 });
    expect(onClickOutside).not.toHaveBeenCalled();
  });

  it("should not call the callback when the click lands inside a portaled child", () => {
    const onClickOutside = vi.fn();
    const c = renderOwner(onClickOutside);
    firePointerDown(c.getByText("Portaled"), { x: 200, y: 200 });
    expect(onClickOutside).not.toHaveBeenCalled();
  });
});
