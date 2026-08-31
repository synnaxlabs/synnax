// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, deep } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Form } from "@/form";
import { type Config, configZ } from "@/schematic/node/general/button/config";
import { ButtonForm } from "@/schematic/node/general/button/Form";
import { Button } from "@/schematic/node/general/button/Primitive";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const getButton = (container: HTMLElement): HTMLElement => {
  const el = container.querySelector<HTMLElement>(".pluto-btn");
  if (el == null) throw new Error("expected a button element");
  return el;
};

describe("button symbol", () => {
  it("should carry the symbol-colored + symbol-button classes and set the source color", () => {
    // The bg/border/text vars are mapped to the display/contrast vars in button.css;
    // jsdom cannot compute them, so we assert the marker classes and the source var.
    const { container } = render(<Button color="#ff0000" />);
    const btn = getButton(container);
    const cls = btn.getAttribute("class") ?? "";
    expect(cls).toContain("pluto-symbol-colored");
    expect(cls).toContain("pluto-symbol-button");
    expect(btn.style.getPropertyValue("--pluto-symbol-color")).toBe("255, 0, 0, 1");
  });

  it("should not engage the base button's concrete-color JS path", () => {
    const { container } = render(<Button color="#ff0000" />);
    const btn = getButton(container);
    // The color is not forwarded, so the base button never sets its own color var.
    expect(btn.getAttribute("class")).not.toContain("pluto-btn--custom-color");
    expect(btn.style.getPropertyValue("--pluto-btn-color")).toBe("");
  });

  it("should carry the alpha channel so a translucent button stays translucent", () => {
    const { container } = render(<Button color={[255, 0, 0, 0.5]} />);
    expect(getButton(container).style.getPropertyValue("--pluto-symbol-color")).toBe(
      "255, 0, 0, 0.5",
    );
  });

  it("should leave the source color unset for the ZERO sentinel", () => {
    const { container } = render(<Button color={color.ZERO} />);
    expect(getButton(container).style.getPropertyValue("--pluto-symbol-color")).toBe(
      "",
    );
  });

  describe("handler routing", () => {
    it("should actuate fire mode through onClick, not the raw handlers", () => {
      const onClick = vi.fn();
      const onMouseUp = vi.fn();
      const { container } = render(
        <Button mode="fire" onClick={onClick} onMouseUp={onMouseUp} />,
      );
      const btn = getButton(container);
      fireEvent.mouseUp(btn);
      expect(onMouseUp).not.toHaveBeenCalled();
      fireEvent.click(btn);
      expect(onClick).toHaveBeenCalledTimes(1);
    });

    it("should actuate momentary mode on raw press and release", () => {
      const onClick = vi.fn();
      const onMouseDown = vi.fn();
      const onMouseUp = vi.fn();
      const { container } = render(
        <Button
          mode="momentary"
          onClick={onClick}
          onMouseDown={onMouseDown}
          onMouseUp={onMouseUp}
        />,
      );
      const btn = getButton(container);
      fireEvent.mouseDown(btn);
      expect(onMouseDown).toHaveBeenCalledTimes(1);
      fireEvent.mouseUp(btn);
      expect(onMouseUp).toHaveBeenCalledTimes(1);
      fireEvent.click(btn);
      expect(onClick).not.toHaveBeenCalled();
    });

    it("should actuate an undelayed pulse on the raw press edge", () => {
      const onMouseDown = vi.fn();
      const { container } = render(<Button mode="pulse" onMouseDown={onMouseDown} />);
      fireEvent.mouseDown(getButton(container));
      expect(onMouseDown).toHaveBeenCalledTimes(1);
    });

    describe("activation delay", () => {
      beforeEach(() => {
        vi.useFakeTimers();
      });

      afterEach(() => {
        vi.useRealTimers();
      });

      it("should swallow a fire-mode click shorter than the delay", () => {
        const onClick = vi.fn();
        const { container } = render(
          <Button mode="fire" onClick={onClick} onClickDelay={500} />,
        );
        const btn = getButton(container);
        fireEvent.mouseDown(btn);
        fireEvent.mouseUp(document);
        fireEvent.click(btn);
        vi.advanceTimersByTime(1000);
        expect(onClick).not.toHaveBeenCalled();
      });

      it("should fire after the delay while the button stays held", () => {
        const onClick = vi.fn();
        const { container } = render(
          <Button mode="fire" onClick={onClick} onClickDelay={500} />,
        );
        fireEvent.mouseDown(getButton(container));
        vi.advanceTimersByTime(499);
        expect(onClick).not.toHaveBeenCalled();
        vi.advanceTimersByTime(1);
        expect(onClick).toHaveBeenCalledTimes(1);
      });

      it("should gate a delayed pulse behind the hold", () => {
        const onMouseDown = vi.fn();
        const { container } = render(
          <Button mode="pulse" onMouseDown={onMouseDown} onClickDelay={500} />,
        );
        const btn = getButton(container);
        fireEvent.mouseDown(btn);
        expect(onMouseDown).not.toHaveBeenCalled();
        vi.advanceTimersByTime(500);
        expect(onMouseDown).toHaveBeenCalledTimes(1);
      });

      it("should swallow a pulse click shorter than the delay", () => {
        const onMouseDown = vi.fn();
        const { container } = render(
          <Button mode="pulse" onMouseDown={onMouseDown} onClickDelay={500} />,
        );
        const btn = getButton(container);
        fireEvent.mouseDown(btn);
        fireEvent.mouseUp(document);
        vi.advanceTimersByTime(1000);
        expect(onMouseDown).not.toHaveBeenCalled();
      });

      it("should ignore the delay for momentary mode", () => {
        const onMouseDown = vi.fn();
        const { container } = render(
          <Button mode="momentary" onMouseDown={onMouseDown} onClickDelay={500} />,
        );
        const btn = getButton(container);
        fireEvent.mouseDown(btn);
        expect(onMouseDown).toHaveBeenCalledTimes(1);
        expect(btn.style.getPropertyValue("--pluto-btn-delay")).toBe("");
      });
    });
  });
});

// A config from before the size control existed: no size key.
const LEGACY_CONFIG: Config = {
  variant: "button",
  orientation: "left",
  color: "#000000",
  label: { label: "Button", level: "h5", orientation: "top" },
  mode: "fire",
};

const SynnaxWrapper = createSynnaxWrapper({ client: null });

const FormWrapper = ({ children }: PropsWithChildren): ReactElement => {
  const methods = Form.use<typeof configZ>({
    values: deep.copy(LEGACY_CONFIG),
    schema: configZ,
  });
  return (
    <SynnaxWrapper>
      <Form.Form<typeof configZ> {...methods}>{children}</Form.Form>
    </SynnaxWrapper>
  );
};

describe("ButtonForm", () => {
  it("should show the size field with medium selected for a config without a size key", () => {
    const { getByText } = render(
      <FormWrapper>
        <ButtonForm />
      </FormWrapper>,
    );
    expect(getByText("Size")).toBeDefined();
    expect(getByText("M").closest("button")?.classList).toContain("pluto--selected");
  });

  it("should not render the label size field", () => {
    const { queryByText } = render(
      <FormWrapper>
        <ButtonForm />
      </FormWrapper>,
    );
    expect(queryByText("Label size")).toBeNull();
  });
});
