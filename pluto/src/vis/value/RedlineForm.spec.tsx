// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { color, deep } from "@synnaxlabs/x";
import { fireEvent, renderHook, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Color } from "@/color";
import { renderForm } from "@/testutil";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { Value } from "@/vis/value";

const client = createTestClient();

const schema = z.object({ redline: color.scaleZ });
const path = deep.path<z.infer<typeof schema>>();

describe("Value.RedlineForm", () => {
  it("should color the band after warning and error with the first series color", async () => {
    const wrapper = await createAsyncSynnaxWrapper({ client });
    const { result } = renderHook(() => Color.useSeriesValues(), { wrapper });
    await waitFor(() => expect(result.current.length).toBeGreaterThan(0));
    const { value } = renderForm(
      <Value.RedlineForm path={path.redline} />,
      { redline: { bands: [], smooth: false } },
      schema,
      wrapper,
    );
    const add = screen.getByRole("button", { name: "Add band" });
    for (let i = 0; i < 3; i++) fireEvent.click(add);
    await waitFor(() => {
      const bands = color.scaleZ.shape.bands.parse(value(path.redline.bands));
      expect(bands).toHaveLength(3);
      expect(color.equals(bands[2].color, result.current[0])).toBe(true);
    });
  });
});
