// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Form } from "@/form";
import { type Config, configZ } from "@/schematic/node/general/button/config";
import { ButtonForm } from "@/schematic/node/general/button/Form";
import { Button } from "@/schematic/node/general/button/Primitive";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const getButton = (container: HTMLElement): HTMLElement => {
  const el = container.querySelector<HTMLElement>("button");
  if (el == null) throw new Error("expected a button element");
  return el;
};

describe("button symbol", () => {
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

    it("should ignore a secondary-button press in momentary mode", () => {
      const onMouseDown = vi.fn();
      const onMouseUp = vi.fn();
      const { container } = render(
        <Button mode="momentary" onMouseDown={onMouseDown} onMouseUp={onMouseUp} />,
      );
      const btn = getButton(container);
      fireEvent.mouseDown(btn, { button: 2 });
      fireEvent.mouseUp(btn, { button: 2 });
      expect(onMouseDown).not.toHaveBeenCalled();
      expect(onMouseUp).not.toHaveBeenCalled();
    });

    it("should ignore a secondary-button press for an undelayed pulse", () => {
      const onMouseDown = vi.fn();
      const { container } = render(<Button mode="pulse" onMouseDown={onMouseDown} />);
      fireEvent.mouseDown(getButton(container), { button: 2 });
      expect(onMouseDown).not.toHaveBeenCalled();
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

      it("should ignore a secondary-button hold in fire mode", () => {
        const onClick = vi.fn();
        const { container } = render(
          <Button mode="fire" onClick={onClick} onClickDelay={500} />,
        );
        fireEvent.mouseDown(getButton(container), { button: 2 });
        vi.advanceTimersByTime(1000);
        expect(onClick).not.toHaveBeenCalled();
      });

      it("should ignore a secondary-button hold for a delayed pulse", () => {
        const onMouseDown = vi.fn();
        const { container } = render(
          <Button mode="pulse" onMouseDown={onMouseDown} onClickDelay={500} />,
        );
        fireEvent.mouseDown(getButton(container), { button: 2 });
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
      });
    });
  });
});

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
    expect(getByText("M").closest("button")?.getAttribute("aria-pressed")).toBe("true");
  });

  it("should not render the label size and direction fields", () => {
    const { queryByText } = render(
      <FormWrapper>
        <ButtonForm />
      </FormWrapper>,
    );
    expect(queryByText("Label size")).toBeNull();
    expect(queryByText("Label direction")).toBeNull();
  });
});
