// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import {
  type PropsWithChildren,
  type ReactElement,
  useImperativeHandle,
  useRef,
} from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Form as Base } from "@/form";
import { Form } from "@/schematic/node/common/form";

const schema = z.object({ scale: z.number() });

interface FormHandle {
  get: () => number;
  set: (v: number) => void;
}

interface HostProps extends PropsWithChildren {
  initial?: number;
  ref?: React.Ref<FormHandle>;
}

// Mounts a Form with a single `scale` field and exposes get/set via a ref so
// tests can drive the form state and observe the underlying value the
// ScaleField writes back.
const Host = ({ initial = 1, children, ref }: HostProps): ReactElement => {
  const methods = Base.use({ values: { scale: initial }, schema });
  const inner = useRef<FormHandle>({
    get: () => methods.get<number>("scale").value,
    set: (v) => methods.set("scale", v),
  });
  useImperativeHandle(ref, () => ({
    get: () => methods.get<number>("scale").value,
    set: (v) => methods.set("scale", v),
  }));
  inner.current = {
    get: () => methods.get<number>("scale").value,
    set: (v) => methods.set("scale", v),
  };
  return <Base.Form<typeof schema> {...methods}>{children}</Base.Form>;
};

const findInput = (container: HTMLElement): HTMLInputElement =>
  container.querySelector('input[type="text"]') as HTMLInputElement;

describe("Form.ScaleField", () => {
  describe("display value", () => {
    it("should display the underlying decimal scaled by 100", () => {
      const { container } = render(
        <Host initial={0.5}>
          <Form.ScaleField path="scale" />
        </Host>,
      );
      expect(findInput(container).value).toBe("50");
    });

    it("should round the displayed value to the nearest integer", () => {
      const { container } = render(
        <Host initial={1.234}>
          <Form.ScaleField path="scale" />
        </Host>,
      );
      // Math.round(1.234 * 100) = 123
      expect(findInput(container).value).toBe("123");
    });

    it("should display 100 for a unit decimal value", () => {
      const { container } = render(
        <Host initial={1}>
          <Form.ScaleField path="scale" />
        </Host>,
      );
      expect(findInput(container).value).toBe("100");
    });

    it("should display the lower bound when the underlying value is at it", () => {
      const { container } = render(
        <Host initial={0.05}>
          <Form.ScaleField path="scale" />
        </Host>,
      );
      expect(findInput(container).value).toBe("5");
    });
  });

  describe("write-back", () => {
    it("should divide the percentage input by 100 when the user commits", () => {
      const ref: { current: FormHandle | null } = { current: null };
      const { container } = render(
        <Host initial={1} ref={ref}>
          <Form.ScaleField path="scale" />
        </Host>,
      );
      const input = findInput(container);
      fireEvent.change(input, { target: { value: "250" } });
      fireEvent.blur(input);
      expect(ref.current?.get()).toBe(2.5);
    });

    it("should clamp values at or above the upper bound (1000) before writing", () => {
      const ref: { current: FormHandle | null } = { current: null };
      const { container } = render(
        <Host initial={1} ref={ref}>
          <Form.ScaleField path="scale" />
        </Host>,
      );
      const input = findInput(container);
      fireEvent.change(input, { target: { value: "5000" } });
      fireEvent.blur(input);
      // 5000 clamps to the 1000 percent maximum -> 10 decimal.
      expect(ref.current?.get()).toBe(10);
    });

    it("should clamp values below the lower bound (5) before writing", () => {
      const ref: { current: FormHandle | null } = { current: null };
      const { container } = render(
        <Host initial={1} ref={ref}>
          <Form.ScaleField path="scale" />
        </Host>,
      );
      const input = findInput(container);
      fireEvent.change(input, { target: { value: "1" } });
      fireEvent.blur(input);
      // 1 clamped up to 5 -> 0.05
      expect(ref.current?.get()).toBe(0.05);
    });

    it("should round the underlying value to two decimal places", () => {
      const ref: { current: FormHandle | null } = { current: null };
      const { container } = render(
        <Host initial={1} ref={ref}>
          <Form.ScaleField path="scale" />
        </Host>,
      );
      const input = findInput(container);
      // 33 / 100 = 0.33 (already 2 dp); 333 / 100 = 3.33; 247 / 100 = 2.47
      fireEvent.change(input, { target: { value: "247" } });
      fireEvent.blur(input);
      expect(ref.current?.get()).toBe(2.47);
    });
  });

  describe("round-trip", () => {
    it("should preserve a 1.5 decimal value across display and re-commit", () => {
      const ref: { current: FormHandle | null } = { current: null };
      const { container, rerender } = render(
        <Host initial={1.5} ref={ref}>
          <Form.ScaleField path="scale" />
        </Host>,
      );
      const input = findInput(container);
      expect(input.value).toBe("150");
      fireEvent.change(input, { target: { value: "150" } });
      fireEvent.blur(input);
      expect(ref.current?.get()).toBe(1.5);
      rerender(
        <Host initial={1.5} ref={ref}>
          <Form.ScaleField path="scale" />
        </Host>,
      );
      // After a rerender, display still reads 150.
      expect(findInput(container).value).toBe("150");
    });
  });

  describe("layout", () => {
    it("should render with the Scale label", () => {
      const { container } = render(
        <Host initial={1}>
          <Form.ScaleField path="scale" />
        </Host>,
      );
      expect(container.textContent).toContain("Scale");
    });

    it("should append a percent sign as the trailing content", () => {
      const { container } = render(
        <Host initial={1}>
          <Form.ScaleField path="scale" />
        </Host>,
      );
      // The end content "%" is rendered alongside the input as an addon.
      expect(container.textContent).toContain("%");
    });
  });
});
