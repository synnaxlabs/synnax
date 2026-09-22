// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { fireEvent, render, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { Input } from "@/input";
import { mockBoundingClientRect } from "@/testutil/dom";
import { Triggers } from "@/triggers";

const local = (...args: [number, number, number, number, number]): number =>
  new TimeStamp(new Date(...args)).nanoseconds;

const VALUE = local(2026, 7, 23, 14, 5);
const HOUR = Number(TimeSpan.HOUR.valueOf());

const open = (container: HTMLElement): HTMLInputElement => {
  const trigger = container.querySelector(".pluto-time-editor__trigger");
  if (trigger == null) throw new Error("no time editor trigger");
  fireEvent.click(trigger);
  return screen.getByRole<HTMLInputElement>("textbox");
};

const type = (field: HTMLInputElement, text: string): void => {
  fireEvent.change(field, { target: { value: text } });
};

describe("Input.DateTime", () => {
  beforeAll(() => {
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
  });

  it("should open the editor on the value in a fixed local layout", () => {
    const { container } = render(<Input.DateTime value={VALUE} onChange={vi.fn()} />);
    expect(open(container).value).toBe("2026-08-23 14:05:00");
  });

  it("should forward unlisted props to the trigger", () => {
    const { container } = render(
      <Input.DateTime value={VALUE} onChange={vi.fn()} aria-label="Start" disabled />,
    );
    const trigger = container.querySelector(".pluto-time-editor__trigger");
    expect(trigger?.getAttribute("aria-label")).toBe("Start");
    expect(trigger?.getAttribute("aria-disabled")).toBe("true");
  });

  it("should commit a typed instant on Enter", () => {
    const onChange = vi.fn();
    const { container } = render(<Input.DateTime value={VALUE} onChange={onChange} />);
    const field = open(container);
    type(field, "2026-08-23 15:30");
    fireEvent.keyDown(field, { key: "Enter" });
    expect(onChange).toHaveBeenCalledWith(local(2026, 7, 23, 15, 30));
    expect(screen.queryByRole("textbox")).toBeNull();
  });

  it("should discard an edit on Escape", () => {
    const onChange = vi.fn();
    const { container } = render(<Input.DateTime value={VALUE} onChange={onChange} />);
    const field = open(container);
    type(field, "2026-08-23 15:30");
    fireEvent.keyDown(field, { key: "Escape" });
    expect(onChange).not.toHaveBeenCalled();
    expect(screen.queryByRole("textbox")).toBeNull();
  });

  it("should discard an edit on Escape when triggers are listening", () => {
    const onChange = vi.fn();
    const { container } = render(
      <Triggers.Provider>
        <Input.DateTime value={VALUE} onChange={onChange} />
      </Triggers.Provider>,
    );
    const field = open(container);
    type(field, "2026-08-23 15:30");
    fireEvent.keyDown(field, { key: "Escape", code: "Escape" });
    expect(onChange).not.toHaveBeenCalled();
    expect(screen.queryByRole("textbox")).toBeNull();
  });

  it("should commit an edit when the trigger closes the editor", () => {
    const onChange = vi.fn();
    const { container } = render(<Input.DateTime value={VALUE} onChange={onChange} />);
    const field = open(container);
    type(field, "2026-08-23 15:30");
    fireEvent.click(container.querySelector(".pluto-time-editor__trigger") as Element);
    expect(onChange).toHaveBeenCalledWith(local(2026, 7, 23, 15, 30));
  });

  it("should not commit an unedited value finer than a microsecond", () => {
    const onChange = vi.fn();
    const { container } = render(
      <Input.DateTime value={VALUE + 512} onChange={onChange} />,
    );
    fireEvent.keyDown(open(container), { key: "Enter" });
    expect(onChange).not.toHaveBeenCalled();
  });

  it("should nudge the unit under the caret with the arrow keys", () => {
    const { container } = render(<Input.DateTime value={VALUE} onChange={vi.fn()} />);
    const field = open(container);
    field.setSelectionRange(12, 12);
    fireEvent.keyDown(field, { key: "ArrowUp" });
    expect(field.value).toBe("2026-08-23 15:05:00");
  });

  it("should walk the readings of a phrase with the arrow keys", () => {
    const onChange = vi.fn();
    const { container } = render(<Input.DateTime value={VALUE} onChange={onChange} />);
    const field = open(container);
    type(field, "2h");
    fireEvent.keyDown(field, { key: "ArrowDown" });
    const before = TimeStamp.now().nanoseconds;
    fireEvent.keyDown(field, { key: "Enter" });
    const committed = onChange.mock.calls[0][0] as number;
    // The second reading of a bare duration is that long ago.
    expect(Math.abs(before - 2 * HOUR - committed)).toBeLessThan(
      Number(TimeSpan.SECOND.valueOf()),
    );
  });

  it("should read a bare duration from the other end of the range first", () => {
    const onChange = vi.fn();
    const end = local(2026, 7, 23, 18, 0);
    const { container } = render(
      <Input.DateTime
        value={VALUE}
        onChange={onChange}
        bound="start"
        anchors={{ end }}
      />,
    );
    const field = open(container);
    type(field, "2h");
    fireEvent.keyDown(field, { key: "Enter" });
    expect(onChange).toHaveBeenCalledWith(end - 2 * HOUR);
  });

  it("should say when a typed date does not exist", () => {
    const { container } = render(<Input.DateTime value={VALUE} onChange={vi.fn()} />);
    type(open(container), "2026-02-31");
    expect(screen.getByText("Not a time")).toBeTruthy();
  });

  it("should commit an edit made after a reading was clicked", () => {
    const onChange = vi.fn();
    const { container } = render(<Input.DateTime value={VALUE} onChange={onChange} />);
    type(open(container), "2026-08-23 15:30");
    fireEvent.click(screen.getByText("local time"));
    expect(onChange).toHaveBeenCalledTimes(1);
    const field = open(container);
    type(field, "2026-08-23 16:45");
    fireEvent.keyDown(field, { key: "Enter" });
    expect(onChange).toHaveBeenLastCalledWith(local(2026, 7, 23, 16, 45));
  });

  it("should commit an action and close", () => {
    const onChange = vi.fn();
    const parent = { start: local(2026, 7, 20, 9, 0), end: local(2026, 7, 21, 9, 0) };
    const { container } = render(
      <Input.DateTime value={VALUE} onChange={onChange} anchors={{ parent }} />,
    );
    open(container);
    fireEvent.click(screen.getByText("Parent start"));
    expect(onChange).toHaveBeenCalledWith(parent.start);
    expect(screen.queryByRole("textbox")).toBeNull();
  });

  it("should give the effect the value each reading and action would commit", () => {
    const parent = { start: local(2026, 7, 20, 9, 0), end: local(2026, 7, 21, 9, 0) };
    const { container } = render(
      <Input.DateTime
        value={VALUE}
        onChange={vi.fn()}
        anchors={{ parent }}
        effect={({ candidate }) => <span data-testid="effect">{candidate}</span>}
      />,
    );
    type(open(container), "2026-08-23 15:30");
    const candidates = screen
      .getAllByTestId("effect")
      .map((e) => Number(e.textContent));
    expect(candidates).toContain(local(2026, 7, 23, 15, 30));
    expect(candidates).toContain(parent.start);
    expect(candidates).toContain(parent.end);
  });

  it("should stay open on Enter when the text is not a time", () => {
    const onChange = vi.fn();
    const { container } = render(<Input.DateTime value={VALUE} onChange={onChange} />);
    const field = open(container);
    type(field, "tomorow 3pm-ish");
    fireEvent.keyDown(field, { key: "Enter" });
    expect(screen.getByRole("textbox")).toBe(field);
    expect(onChange).not.toHaveBeenCalled();
  });

  it("should commit the empty value when the field is cleared", () => {
    const onChange = vi.fn();
    const empty = TimeStamp.MAX.nanoseconds;
    const { container } = render(
      <Input.DateTime value={VALUE} onChange={onChange} emptyValue={empty} />,
    );
    const field = open(container);
    type(field, "");
    fireEvent.keyDown(field, { key: "Enter" });
    expect(onChange).toHaveBeenCalledWith(empty);
  });

  it("should show a value later than the empty value", () => {
    render(
      <Input.DateTime
        value={VALUE}
        onChange={vi.fn()}
        emptyValue={local(2000, 0, 1, 0, 0)}
        placeholder="Set a time"
      />,
    );
    expect(screen.queryByText("Set a time")).toBeNull();
  });
});
