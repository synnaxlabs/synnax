// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { TimeSpan } from "@synnaxlabs/x";
import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LinePlot } from "@/platform/lineplot";
import { Session } from "@/session";
import { createConsoleWrapper } from "@/testutil";

const client = createTestClient();

const STATIC: Session.Range.StaticState = {
  variant: "static",
  key: "5c1d3e7f-2a4b-4c6d-8e0f-1a3b5c7d9e0f",
  name: "Hot fire",
  timeRange: { start: 1_000_000_000, end: 3_000_000_000 },
};

const renderGetDefaultRanges = async (
  ranges: Session.Range.State[],
  selected?: string,
) => {
  const { wrapper } = await createConsoleWrapper({
    client,
    preloadedState: {
      [Session.Range.SLICE_NAME]: {
        ...Session.Range.ZERO_SLICE_STATE,
        ranges,
        selected,
      },
    },
  });
  return renderHook(LinePlot.useGetDefaultRanges, { wrapper }).result.current;
};

describe("lineplot ranges", () => {
  describe("fromSession", () => {
    it("should copy a static range's window and name into the plot", () => {
      expect(LinePlot.fromSession(STATIC)).toEqual({
        variant: "static",
        key: STATIC.key,
        name: "Hot fire",
        start: "1000000000",
        end: "3000000000",
      });
    });

    it("should refer to a persisted range by key", () => {
      const key = "8d2e4f6a-1b3c-4d5e-9f7a-2b4c6d8e0f1a";
      expect(LinePlot.fromSession({ variant: "persisted", key })).toEqual({
        variant: "persisted",
        key,
      });
    });
  });

  describe("useGetDefaultRanges", () => {
    it("should give a 30s rolling window when no range is selected", async () => {
      const getDefaultRanges = await renderGetDefaultRanges([STATIC]);
      expect(getDefaultRanges()).toEqual({
        x1: { rolling: Number(TimeSpan.seconds(30)), ranges: [] },
      });
    });

    it("should give the selected range when one is selected", async () => {
      const getDefaultRanges = await renderGetDefaultRanges([STATIC], STATIC.key);
      expect(getDefaultRanges()).toEqual({
        x1: { ranges: [LinePlot.fromSession(STATIC)] },
      });
    });
  });
});
