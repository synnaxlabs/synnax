// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { deep } from "@synnaxlabs/x";
import { fireEvent, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Form } from "@/schematic/node/common/form";
import { renderForm } from "@/testutil";

const path = deep.path<schematic.SetpointNodeConfig>();

const SAVED = schematic.setpointNodeConfigZ.parse({
  variant: "setpoint",
  size: "large",
  level: "h5",
});

describe("Form.ScaledSizeField", () => {
  it("should apply the standard level when the selected size is clicked", async () => {
    const { getByText, value } = renderForm(
      <Form.ScaledSizeField />,
      SAVED,
      schematic.setpointNodeConfigZ,
    );
    fireEvent.click(getByText("L"));
    await waitFor(() => {
      expect(value(path.size)).toBe("large");
      expect(value(path.level)).toBe("h3");
    });
  });

  it("should set the standard level with the size", async () => {
    const { getByText, value } = renderForm(
      <Form.ScaledSizeField />,
      SAVED,
      schematic.setpointNodeConfigZ,
    );
    fireEvent.click(getByText("XL"));
    await waitFor(() => {
      expect(value(path.size)).toBe("huge");
      expect(value(path.level)).toBe("h2");
    });
  });
});
