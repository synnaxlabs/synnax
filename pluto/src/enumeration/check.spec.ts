// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type enumeration } from "@synnaxlabs/client";
import { theme } from "@synnaxlabs/lyra/theme";
import { color } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { check, type ResolveColor } from "@/enumeration/check";

const LIGHT = theme.themeZ.parse(theme.SYNNAX_LIGHT);

const resolve: ResolveColor = ({ color: ref }) =>
  ref?.type === "fixed" ? { light: ref.value } : undefined;

const createEntry = (
  name: string,
  value: number,
  hex?: string,
  abnormal = false,
): enumeration.Entry => ({
  key: name,
  name,
  value,
  abnormal,
  ...(hex != null && { color: { type: "fixed", value: color.construct(hex) } }),
});

const ALARM = color.hex(LIGHT.colors.error.z);

describe("check", () => {
  it("should warn about an entry with no name", () => {
    const entry = createEntry(" ", 0);
    expect(check(entry, [entry], resolve)).toEqual([
      "It has no name, so only its color shows it.",
    ]);
  });

  it("should warn about an entry whose name another entry has", () => {
    const a = createEntry("Open", 0);
    const b = { ...createEntry("Open", 1), key: "other" };
    expect(check(a, [a, b], resolve)).toEqual([
      "Another entry has its name, so only color tells them apart.",
    ]);
  });

  it("should warn about a color that is hard to see on the canvas", () => {
    const entry = createEntry("Off", 0, "#ffffff");
    expect(check(entry, [entry], resolve)).toContain(
      "Its color is hard to see in the light theme.",
    );
  });

  it("should name the entry whose color is almost the same", () => {
    const a = createEntry("Open", 0, "#2e7d32");
    const b = createEntry("Running", 1, "#2e7e32");
    expect(check(a, [a, b], resolve)).toContain(
      "Its color is almost the color of Running.",
    );
  });

  it("should warn about an alarm color on an entry that is not abnormal", () => {
    const entry = createEntry("Locked out", 0, ALARM);
    expect(check(entry, [entry], resolve)).toContain(
      "It has an alarm color, but it is not abnormal.",
    );
  });

  it("should not warn about an alarm color on an abnormal entry", () => {
    const entry = createEntry("Fault", 0, ALARM, true);
    expect(check(entry, [entry], resolve)).not.toContain(
      "It has an alarm color, but it is not abnormal.",
    );
  });

  it("should not warn about an alarm color when no entry can be abnormal", () => {
    const entry = createEntry("Locked out", 0, ALARM);
    expect(check(entry, [entry], resolve, { abnormalDisabled: true })).not.toContain(
      "It has an alarm color, but it is not abnormal.",
    );
  });

  it("should not check the color of an entry whose symbol picks it", () => {
    const entry = createEntry("Off", 0);
    expect(check(entry, [entry], resolve)).toEqual([]);
  });
});
