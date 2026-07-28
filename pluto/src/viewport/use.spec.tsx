// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Triggers } from "@/triggers";
import { use } from "@/viewport/use";

const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
  <Triggers.Provider>{children}</Triggers.Provider>
);

describe("viewport/use", () => {
  describe("wheel zoom", () => {
    // Regression: the window-level wheel handler measured the canvas before checking
    // that the event targeted it, forcing a layout read per mounted viewport on every
    // wheel event anywhere in the app.
    it("should not measure the canvas for wheel events outside it", () => {
      const { result } = renderHook(() => use({}), { wrapper: Wrapper });
      const canvas = document.createElement("div");
      const outside = document.createElement("div");
      document.body.append(canvas, outside);
      act(() => {
        result.current.ref.current = canvas;
      });
      const measure = vi.spyOn(canvas, "getBoundingClientRect");
      outside.dispatchEvent(new WheelEvent("wheel", { bubbles: true, deltaY: 1 }));
      expect(measure).not.toHaveBeenCalled();
      canvas.dispatchEvent(new WheelEvent("wheel", { bubbles: true, deltaY: 1 }));
      expect(measure).toHaveBeenCalled();
      canvas.remove();
      outside.remove();
    });
  });
});
