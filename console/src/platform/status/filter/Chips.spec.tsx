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
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Status } from "@/platform/status";
import { View } from "@/platform/view";
import { enableEditing } from "@/platform/view/testutil";
import { createConsoleWrapper, findTagCloseButton } from "@/testutil";

const client = createTestClient();

const schema = z.object({
  query: z.object({ variants: z.array(z.string()) }),
});

const Fixture = ({ variants }: { variants: status.Variant[] }): ReactElement => {
  const methods = Form.use({ schema, values: { query: { variants } } });
  return (
    <View.Frame resourceType="status" icon="Status">
      <Form.Form<typeof schema> {...methods}>
        <Status.Filter.Chips />
        <Form.Field<status.Variant[]> path="query.variants" showLabel={false}>
          {({ value }) => <span>{`value:${value.join(",")}`}</span>}
        </Form.Field>
      </Form.Form>
    </View.Frame>
  );
};
Fixture.displayName = "Fixture";

describe("Status.Filter.Chips", () => {
  it("should remove the variant from the filter query when its chip is closed", async () => {
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Fixture variants={["error", "success"]} />, { wrapper });
    await waitFor(() => expect(screen.getByText("Error")).toBeTruthy());
    await enableEditing();
    const close = await waitFor(() => findTagCloseButton("Error"));
    fireEvent.click(close);
    await waitFor(() => expect(screen.getByText("value:success")).toBeTruthy());
    expect(screen.queryByText("Error")).toBeNull();
    expect(screen.getByText("Success")).toBeTruthy();
  });

  it("should render nothing when no variants are selected", async () => {
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Fixture variants={[]} />, { wrapper });
    await waitFor(() => expect(screen.getByText("All Statuses")).toBeTruthy());
    expect(screen.queryByText("Variants")).toBeNull();
  });
});
