// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { TimeSpan } from "@synnaxlabs/x";
import { act, fireEvent, render } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Device } from "@/device";

const DELAY = TimeSpan.milliseconds(10);

const LOCATION_RECT = {
  left: 100,
  top: 100,
  right: 200,
  bottom: 120,
  width: 100,
  height: 20,
  x: 100,
  y: 100,
  toJSON: () => ({}),
} as DOMRect;

const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
  <Triggers.Provider>
    <Tooltip.Config delay={DELAY}>{children}</Tooltip.Config>
  </Triggers.Provider>
);

const hoverPastDelay = (text: HTMLElement): void => {
  const location = text.closest<HTMLElement>(".pluto-device-location");
  if (location == null) throw new Error("No location around the text");
  location.getBoundingClientRect = () => LOCATION_RECT;
  fireEvent.pointerOver(text, { pointerType: "mouse" });
  act(() => {
    vi.advanceTimersByTime(DELAY.milliseconds * 2);
  });
};

const getTooltip = (): HTMLElement | null =>
  document.querySelector<HTMLElement>(".lyra-tooltip");

describe("Device location", () => {
  it("should show the whole value without a format", () => {
    const c = render(<Device.Location value="cDAQ1Mod1" />);
    expect(c.getByText("cDAQ1Mod1")).toBeTruthy();
  });

  it("should show only the last part of a slot", () => {
    const c = render(
      <Device.Location value="\Device\NPF_{6A1B}.Slot 3" format={{ shape: "slot" }} />,
    );
    expect(c.getByText("Slot 3")).toBeTruthy();
    expect(c.queryByText(/NPF/)).toBeNull();
  });

  it("should hide each scheme that the format hides", () => {
    const c = render(
      <Device.Location
        value="opc.tcp://10.0.0.5:4840"
        format={{ shape: "address", hiddenSchemes: ["opc.tcp://"] }}
      />,
    );
    expect(c.getByText("10.0.0.5")).toBeTruthy();
    expect(c.getByText(":4840")).toBeTruthy();
    expect(c.queryByText(/opc\.tcp/)).toBeNull();
  });

  it("should keep the whole of a location without a host in view", () => {
    const value = "/dev/ttyUSB0";
    const c = render(<Device.Location value={value} format={{ shape: "address" }} />);
    const location = c.getByText(value).closest<HTMLElement>(".pluto-device-location");
    expect(location?.style.getPropertyValue("--pluto-location-keep")).toBe(
      `${value.length}`,
    );
  });

  it("should keep a scheme that the format does not hide", () => {
    const c = render(
      <Device.Location
        value="https://api.example.com/v1"
        format={{ shape: "address" }}
      />,
    );
    expect(c.getByText("https://")).toBeTruthy();
    expect(c.getByText("api.example.com")).toBeTruthy();
    expect(c.getByText("/v1")).toBeTruthy();
  });

  it("should keep the brackets of an IPv6 host", () => {
    const c = render(
      <Device.Location value="http://[fe80::1]:8080" format={{ shape: "address" }} />,
    );
    expect(c.getByText("[fe80::1]")).toBeTruthy();
    expect(c.getByText(":8080")).toBeTruthy();
  });

  it("should keep an IPv6 host without brackets whole", () => {
    const c = render(
      <Device.Location value="mqtt://fd00::5/topic" format={{ shape: "address" }} />,
    );
    expect(c.getByText("fd00::5")).toBeTruthy();
    expect(c.getByText("/topic")).toBeTruthy();
  });

  it("should show the first address of a list and a count of the rest", () => {
    const c = render(
      <Device.Location
        value="b1.local:9092, b2.local:9092, b3.local:9092"
        format={{ shape: "list" }}
      />,
    );
    expect(c.getByText("b1.local")).toBeTruthy();
    expect(c.getByText("+2")).toBeTruthy();
    expect(c.queryByText(/b2\.local/)).toBeNull();
  });

  describe("tooltip", () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it("should show the whole location when part of it is hidden", () => {
      const value = "\\Device\\NPF_{6A1B}.Slot 3";
      const c = render(<Device.Location value={value} format={{ shape: "slot" }} />, {
        wrapper: Wrapper,
      });
      hoverPastDelay(c.getByText("Slot 3"));
      expect(getTooltip()?.textContent).toBe(value);
    });

    it("should not show when the whole location shows", () => {
      const c = render(<Device.Location value="cDAQ1Mod1" />, { wrapper: Wrapper });
      hoverPastDelay(c.getByText("cDAQ1Mod1"));
      expect(getTooltip()).toBeNull();
    });
  });
});
