// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Form } from "@synnaxlabs/lyra/form";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Status } from "@/platform/status";
import { View } from "@/platform/view";
import { enableEditing } from "@/platform/view/testutil";
import { createConsoleWrapper, findTagCloseButton } from "@/testutil";

const client = createTestClient();

const schema = z.object({ where: status.whereZ.optional() });

interface FixtureProps {
  variant?: status.Where["variant"];
}

const Fixture = ({ variant }: FixtureProps): ReactElement => {
  const methods = Form.use({
    schema,
    values: { where: variant == null ? undefined : { variant } },
  });
  return (
    <View.Frame resourceType="status" icon="Status">
      <Form.Form<typeof schema> {...methods}>
        <Status.Filter.Chips />
        <Form.Field<status.Where["variant"]> path="where.variant" showLabel={false}>
          {({ value }) => <span>{`value:${value?.in?.join(",") ?? "none"}`}</span>}
        </Form.Field>
      </Form.Form>
    </View.Frame>
  );
};
Fixture.displayName = "Fixture";

describe("Status.Filter.Chips", () => {
  it("should remove the variant from the where when its chip is closed", async () => {
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Fixture variant={{ in: ["error", "success"] }} />, { wrapper });
    await waitFor(() => expect(screen.getByText("Error")).toBeTruthy());
    await enableEditing();
    const close = await waitFor(() => findTagCloseButton("Error"));
    fireEvent.click(close);
    await waitFor(() => expect(screen.getByText("value:success")).toBeTruthy());
    expect(screen.queryByText("Error")).toBeNull();
    expect(screen.getByText("Success")).toBeTruthy();
  });

  it("should drop the variant line when the last chip is closed", async () => {
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Fixture variant={{ in: ["error"] }} />, { wrapper });
    await waitFor(() => expect(screen.getByText("Error")).toBeTruthy());
    await enableEditing();
    fireEvent.click(await waitFor(() => findTagCloseButton("Error")));
    await waitFor(() => expect(screen.getByText("value:none")).toBeTruthy());
  });

  it("should show and remove a variant that the where selects with eq", async () => {
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Fixture variant={{ eq: "error" }} />, { wrapper });
    await waitFor(() => expect(screen.getByText("Error")).toBeTruthy());
    await enableEditing();
    fireEvent.click(await waitFor(() => findTagCloseButton("Error")));
    await waitFor(() => expect(screen.queryByText("Variants")).toBeNull());
  });

  it("should render nothing when no variants are selected", async () => {
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Fixture />, { wrapper });
    await waitFor(() => expect(screen.getByText("All statuses")).toBeTruthy());
    expect(screen.queryByText("Variants")).toBeNull();
  });
});
