// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type label } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Label } from "@/platform/label";
import { createConsoleWrapper } from "@/testutil";

const TIMEOUT = { timeout: 5000 };

const schema = z.object({
  query: z.object({ hasLabels: z.array(z.string()) }),
});

const Fixture = ({ onValue }: { onValue?: (v: label.Key[]) => void }): ReactElement => {
  const methods = Form.use({ schema, values: { query: { hasLabels: [] } } });
  return (
    <Form.Form<typeof schema> {...methods}>
      <Label.Filter.MenuItem />
      <Form.Field<label.Key[]> path="query.hasLabels" showLabel={false}>
        {({ value }) => {
          onValue?.(value);
          return <span data-testid="value">{value.join(",")}</span>;
        }}
      </Form.Field>
    </Form.Form>
  );
};
Fixture.displayName = "Fixture";

const getTrigger = async (): Promise<HTMLElement> =>
  await waitFor(() => {
    const el = document.querySelector<HTMLElement>(".pluto-dialog__trigger");
    if (el == null) throw new Error("select trigger not found");
    return el;
  }, TIMEOUT);

describe("Label.Filter.MenuItem", () => {
  it("should render the label filter select trigger", async () => {
    const client = createTestClient();
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Fixture />, { wrapper });
    expect(await getTrigger()).toBeTruthy();
  });

  it("should add a selected label's key to the query field", async () => {
    const client = createTestClient();
    const name = id.create();
    await client.labels.create({ name, color: "#000000" });
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Fixture />, { wrapper });
    fireEvent.click(await getTrigger());
    const option = await waitFor(() => screen.getByText(name), TIMEOUT);
    fireEvent.click(option);
    await waitFor(
      () => expect(screen.getByTestId("value").textContent).not.toBe(""),
      TIMEOUT,
    );
  });
});
