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
import { deep } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Status } from "@/platform/status";
import { View } from "@/platform/view";
import { createConsoleWrapper } from "@/testutil";

const path = deep.path<z.infer<typeof schema>>();

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
        <Form.Field path={deep.present(path.where).variant} showLabel={false}>
          {({ value }) => <span>{`value:${value?.in?.join(",") ?? "none"}`}</span>}
        </Form.Field>
      </Form.Form>
    </View.Frame>
  );
};
Fixture.displayName = "Fixture";

const renderFixture = async (variant?: status.Where["variant"]): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client });
  render(<Fixture variant={variant} />, { wrapper });
};

describe("Status.Filter.Chips", () => {
  it("should name the variants after is any of", async () => {
    await renderFixture({ in: ["error", "success"] });
    await waitFor(() => expect(screen.getByText("Error, Success")).toBeTruthy());
    expect(screen.getByText("Variants")).toBeTruthy();
    expect(screen.getByText("is any of")).toBeTruthy();
  });

  it("should show a variant that the where selects with eq", async () => {
    await renderFixture({ eq: "error" });
    await waitFor(() => expect(screen.getByText("Error")).toBeTruthy());
    expect(screen.getByText("is")).toBeTruthy();
  });

  it("should drop the variant line when the chip is removed", async () => {
    await renderFixture({ in: ["error", "success"] });
    fireEvent.click(
      await screen.findByRole("button", { name: "Remove Variants filter" }),
    );
    await waitFor(() => expect(screen.getByText("value:none")).toBeTruthy());
    expect(screen.queryByText("Variants")).toBeNull();
  });

  it("should render nothing when no variants are selected", async () => {
    await renderFixture();
    await waitFor(() => expect(screen.getByText("All statuses")).toBeTruthy());
    expect(screen.queryByText("Variants")).toBeNull();
  });
});
