// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label, ranger } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Form } from "@synnaxlabs/lyra/form";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Label } from "@/platform/label";
import { searchAndClickLabel } from "@/platform/label/testutil";
import { createConsoleWrapper, uniqueName } from "@/testutil";

const schema = z.object({ where: ranger.whereZ.optional() });

const Fixture = (): ReactElement => {
  const methods = Form.use({ schema, values: {} });
  return (
    <Form.Form<typeof schema> {...methods}>
      <Label.Filter.MenuItem />
      <Form.Field<label.Key[] | undefined> path="where.labels" showLabel={false}>
        {({ value }) => <span>{`value:${value?.join(",") ?? "none"}`}</span>}
      </Form.Field>
    </Form.Form>
  );
};
Fixture.displayName = "Fixture";

describe("Label.Filter.MenuItem", () => {
  it("should add a selected label's key to the where", async () => {
    const client = createTestClient();
    const name = uniqueName("label");
    const label = await client.labels.create({ name, color: "#000000" });
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Fixture />, { wrapper });
    await searchAndClickLabel(name);
    await waitFor(() => expect(screen.getByText(`value:${label.key}`)).toBeTruthy());
  });

  it("should drop the labels line when the selection empties", async () => {
    const client = createTestClient();
    const name = uniqueName("label");
    const label = await client.labels.create({ name, color: "#000000" });
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Fixture />, { wrapper });
    await searchAndClickLabel(name);
    await waitFor(() => expect(screen.getByText(`value:${label.key}`)).toBeTruthy());
    fireEvent.click(screen.getByText(name));
    await waitFor(() => expect(screen.getByText("value:none")).toBeTruthy());
  });
});
