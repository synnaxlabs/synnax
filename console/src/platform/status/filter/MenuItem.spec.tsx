// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Status } from "@/platform/status";
import { renderWithConsole } from "@/testutil";

const TIMEOUT = { timeout: 5000 };

const schema = z.object({
  query: z.object({ variants: z.array(z.string()) }),
});

const Fixture = (): ReactElement => {
  const methods = Form.use({ schema, values: { query: { variants: [] } } });
  return (
    <Form.Form<typeof schema> {...methods}>
      <Status.Filter.MenuItem />
      <Form.Field<status.Variant[]> path="query.variants" showLabel={false}>
        {({ value }) => <span data-testid="value">{value.join(",")}</span>}
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

describe("Status.Filter.MenuItem", () => {
  it("should render the variant filter select trigger", async () => {
    await renderWithConsole(<Fixture />);
    expect(await getTrigger()).toBeTruthy();
  });

  it("should add the chosen variant to the query field", async () => {
    await renderWithConsole(<Fixture />);
    fireEvent.click(await getTrigger());
    const option = await waitFor(() => screen.getByText("Error"), TIMEOUT);
    fireEvent.click(option);
    await waitFor(() =>
      expect(screen.getByTestId("value").textContent).toContain("error"),
    );
  });
});
