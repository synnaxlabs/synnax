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
  describe("icon", () => {
    it("should return a circle icon when the user has control", () => {
      const { chipIcon } = tooltipMessage(makeStatus("success"));
      expect(chipIcon).toBe(Icon.Circle);
    });

    it("should return a circle icon when the user has absolute control", () => {
      const { chipIcon } = tooltipMessage(
        makeStatus("success", { authority: control.ABSOLUTE_AUTHORITY }),
      );
      expect(chipIcon).toBe(Icon.Circle);
    });

    it("should return a square icon when someone else has control", () => {
      const { chipIcon } = tooltipMessage(makeStatus("error"));
      expect(chipIcon).toBe(Icon.Square);
    });

    it("should return a circle icon when the channel is uncontrolled", () => {
      const { chipIcon } = tooltipMessage(makeStatus("disabled", { valid: true }));
      expect(chipIcon).toBe(Icon.Circle);
    });

    it("should return a circle icon when no channel is connected", () => {
      const { chipIcon } = tooltipMessage(makeStatus("disabled", { valid: false }));
      expect(chipIcon).toBe(Icon.Circle);
    });
  });

  describe("color", () => {
    it("should return the error color when someone else has control", () => {
      const { chipColor } = tooltipMessage(makeStatus("error"));
      expect(chipColor).toBe("var(--pluto-error-z)");
    });

    it("should return the primary color when the user has control", () => {
      const { chipColor } = tooltipMessage(makeStatus("success"));
      expect(chipColor).toBe("var(--pluto-primary-z)");
    });

    it("should return the secondary color for absolute control", () => {
      const { chipColor } = tooltipMessage(
        makeStatus("success", { authority: control.ABSOLUTE_AUTHORITY }),
      );
      expect(chipColor).toBe("var(--pluto-secondary-z)");
    });
  });

  describe("disabled", () => {
    it("should be disabled when no channel is connected", () => {
      const { disabled } = tooltipMessage(makeStatus("disabled", { valid: false }));
      expect(disabled).toBe(true);
    });

    it("should not be disabled when the channel is uncontrolled", () => {
      const { disabled } = tooltipMessage(makeStatus("disabled", { valid: true }));
      expect(disabled).toBeUndefined();
    });
  });
});
