// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { describe, expect, it, vi } from "vitest";

import { Select } from "@/schematic/node/general/select/Primitive";

const OPTIONS = [
  { key: "a", name: "A", value: 7 },
  { key: "b", name: "B", value: 8 },
  { key: "c", name: "C", value: 9 },
];

describe("select symbol draft", () => {
  const isDirty = (container: HTMLElement): boolean =>
    container
      .querySelector(".pluto-select-symbol")
      ?.classList.contains("pluto--dirty") ?? false;

  const sendB = async (onSend: (value: number) => void) => {
    const r = await renderThemed(
      <Select value="a" options={OPTIONS} onSend={onSend} />,
    );
    await r.getByText("A", { exact: true }).click();
    await r.getByText("B", { exact: true }).click();
    await r.getByText("Set", { exact: true }).click();
    return r;
  };

  it("should keep the sent option until the target reports it", async () => {
    const onSend = vi.fn();
    const { container, getByText, rerender } = await sendB(onSend);
    expect(onSend).toHaveBeenCalledExactlyOnceWith(8);
    await expect.element(getByText("A", { exact: true })).not.toBeInTheDocument();
    expect(isDirty(container)).toBe(true);
    await rerender(<Select value="b" options={OPTIONS} onSend={onSend} />);
    expect(isDirty(container)).toBe(false);
  });

  it("should follow the target once it reports another option", async () => {
    const onSend = vi.fn();
    const { container, getByText, rerender } = await sendB(onSend);
    await rerender(<Select value="c" options={OPTIONS} onSend={onSend} />);
    await expect.element(getByText("C", { exact: true })).toBeInTheDocument();
    expect(isDirty(container)).toBe(false);
  });
});
