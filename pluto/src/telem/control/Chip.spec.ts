// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { control } from "@synnaxlabs/client";
import { type status, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";
import { type z } from "zod";

import { Icon } from "@/icon";
import { type control as controlAether } from "@/telem/control/aether";
import { tooltipMessage } from "@/telem/control/Chip";

const makeStatus = (
  variant: status.Variant,
  details: Partial<z.infer<typeof controlAether.chipStatusDetailsZ>> = {},
): status.Status<typeof controlAether.chipStatusDetailsZ> => ({
  key: "test",
  name: "test",
  variant,
  message: "",
  time: TimeStamp.now(),
  details: { authority: undefined, valid: false, ...details },
});

describe("tooltipMessage", () => {
  it("should return controlled style when the user has control", () => {
    const result = tooltipMessage(makeStatus("success"));
    expect(result).toEqual({
      message: "You're in control. Release schematic to release control.",
      chipColor: "var(--pluto-primary-z)",
      chipIcon: Icon.Circle,
    });
  });

  it("should return absolute control style with background", () => {
    const result = tooltipMessage(
      makeStatus("success", { authority: control.ABSOLUTE_AUTHORITY }),
    );
    expect(result).toEqual({
      message: "You have absolute control. Click to release.",
      chipColor: "var(--pluto-secondary-z)",
      chipIcon: Icon.Circle,
      buttonStyle: { background: "var(--pluto-secondary-z-30)" },
    });
  });

  it("should return error style with square icon when someone else has control", () => {
    const result = tooltipMessage(makeStatus("error"));
    expect(result).toEqual({
      message: "Not controlled by you. Click to take absolute control.",
      chipColor: "var(--pluto-error-z)",
      chipIcon: Icon.Square,
    });
  });

  it("should return uncontrolled style when the channel is uncontrolled", () => {
    const result = tooltipMessage(makeStatus("disabled", { valid: true }));
    expect(result).toEqual({
      message: "Uncontrolled. Click to take control.",
      chipColor: "var(--pluto-gray-l12)",
      chipIcon: Icon.Circle,
    });
  });

  it("should return disabled style when no channel is connected", () => {
    const result = tooltipMessage(makeStatus("disabled", { valid: false }));
    expect(result).toEqual({
      message: "No channel connected. This element cannot be controlled.",
      chipColor: "var(--pluto-gray-l7)",
      chipIcon: Icon.Circle,
      disabled: true,
    });
  });

  it("should return error style for an unexpected variant", () => {
    const result = tooltipMessage(makeStatus("info" as status.Variant));
    expect(result).toEqual({
      message: "Unexpected status.",
      chipColor: "var(--pluto-error-z)",
      chipIcon: Icon.Square,
    });
  });
});
