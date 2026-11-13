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
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Status } from "@/platform/status";
import { View } from "@/platform/view";
import { enableEditing } from "@/platform/view/testutil";
import { createConsoleWrapper } from "@/testutil";

const client = createTestClient();

const path = deep.path<z.infer<typeof schema>>();

const schema = z.object({ where: status.whereZ.optional() });

const PROPERTIES: View.Filter.Properties<status.Where> = {
  variant: Status.Filter.VARIANT,
};

const Fixture = ({ where }: { where?: status.Where }): ReactElement => {
  const methods = Form.use({ schema, values: { where } });
  return (
    <View.Frame resourceType="status" icon="Status">
      <Form.Form<typeof schema> {...methods}>
        <View.Filter.Chips properties={PROPERTIES} />
        <Form.Field path={path.where} showLabel={false}>
          {({ value }) => <span>{`where:${JSON.stringify(value) ?? "none"}`}</span>}
        </Form.Field>
      </Form.Form>
    </View.Frame>
  );
};
Fixture.displayName = "Fixture";

const renderFixture = async (where?: status.Where): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client });
  render(<Fixture where={where} />, { wrapper });
};

describe("Status.Filter.VARIANT", () => {
  it("should name the variants after is any of", async () => {
    await renderFixture({ variant: { in: ["error", "warning"] } });
    expect(await screen.findByText("Error, Warning")).toBeTruthy();
    expect(screen.getByText("is any of")).toBeTruthy();
  });

  it("should name one variant after is", async () => {
    await renderFixture({ variant: { eq: "info" } });
    expect(await screen.findByText("Info")).toBeTruthy();
    expect(screen.getByText("is")).toBeTruthy();
  });

  it("should add a variant from the picker below the chip", async () => {
    await renderFixture({ variant: { in: ["error"] } });
    await enableEditing();
    fireEvent.click(await screen.findByRole("button", { name: "Error" }));
    const picker = await screen.findByRole("dialog", { name: "Variants" });
    fireEvent.click(within(picker).getByText("Warning"));
    await waitFor(() =>
      expect(
        screen.getByText(
          `where:${JSON.stringify({ and: [{ variant: { in: ["error", "warning"] } }] })}`,
        ),
      ).toBeTruthy(),
    );
  });
});
