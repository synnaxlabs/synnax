// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { CustomScaleForm } from "@/feature/labjack/task/ReadChannelForms";
import { type Scale } from "@/feature/labjack/task/types";
import { renderInTaskForm } from "@/platform/task/testutil";

const path = deep.path<{ config: { scale: Scale } }>();

describe("CustomScaleForm", () => {
  it("should hold only the fields of the new scale type", async () => {
    const scale = { type: "linear", slope: 2, offset: 1 };
    const { form } = await renderInTaskForm(
      <CustomScaleForm path={path.config.scale} />,
      { values: { config: { scale } } },
    );
    fireEvent.click(await screen.findByText("Linear"));
    fireEvent.click(await screen.findByText("None"));
    await waitFor(() =>
      expect(form.current?.get(path.config.scale).value).toEqual({ type: "none" }),
    );
  });
});
