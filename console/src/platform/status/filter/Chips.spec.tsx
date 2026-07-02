// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type status } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/pluto";
import { render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Status } from "@/platform/status";
import { View } from "@/platform/view";
import { createConsoleWrapper } from "@/testutil";

const TIMEOUT = { timeout: 5000 };

const schema = z.object({
  query: z.object({ variants: z.array(z.string()) }),
});

const Fixture = ({ variants }: { variants: status.Variant[] }): ReactElement => {
  const methods = Form.use({ schema, values: { query: { variants } } });
  return (
    <View.Frame resourceType="status" icon="Status">
      <Form.Form<typeof schema> {...methods}>
        <Status.Filter.Chips />
      </Form.Form>
    </View.Frame>
  );
};
Fixture.displayName = "Fixture";

describe("Status.Filter.Chips", () => {
  it("should render a capitalized tag for each selected variant", async () => {
    const client = createTestClient();
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Fixture variants={["error", "success"]} />, { wrapper });
    await waitFor(() => expect(screen.getByText("Variants")).toBeTruthy(), TIMEOUT);
    expect(screen.getByText("Error")).toBeTruthy();
    expect(screen.getByText("Success")).toBeTruthy();
  });

  it("should render nothing when no variants are selected", async () => {
    const client = createTestClient();
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Fixture variants={[]} />, { wrapper });
    await waitFor(() => expect(screen.getByText("All Statuses")).toBeTruthy(), TIMEOUT);
    expect(screen.queryByText("Variants")).toBeNull();
  });
});
