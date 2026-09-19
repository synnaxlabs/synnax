// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Setpoint } from "@/schematic/node/general/setpoint/Primitive";

describe("setpoint symbol", () => {
  describe("activation delay", () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it("should set on a plain click without a delay", () => {
      const onChange = vi.fn();
      const { getByText } = render(<Setpoint onChange={onChange} />);
      fireEvent.click(getByText("Set"));
      expect(onChange).toHaveBeenCalledWith(0);
    });

    it("should swallow a click shorter than the delay", () => {
      const onChange = vi.fn();
      const { getByText } = render(<Setpoint onChange={onChange} onClickDelay={500} />);
      const btn = getByText("Set");
      fireEvent.mouseDown(btn);
      fireEvent.mouseUp(document);
      fireEvent.click(btn);
      vi.advanceTimersByTime(1000);
      expect(onChange).not.toHaveBeenCalled();
    });

    it("should set after the delay while the button stays held", () => {
      const onChange = vi.fn();
      const { getByText } = render(<Setpoint onChange={onChange} onClickDelay={500} />);
      fireEvent.mouseDown(getByText("Set"));
      vi.advanceTimersByTime(500);
      expect(onChange).toHaveBeenCalledWith(0);
    });
  });
});
