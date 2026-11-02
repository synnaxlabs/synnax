// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Status } from "@/platform/status";
import { findDialogTrigger, renderWithConsole } from "@/testutil";

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
    <Form.Form<typeof schema> {...methods}>
      <Status.Filter.MenuItem />
      <Form.Field<status.Where["variant"]> path="where.variant" showLabel={false}>
        {({ value }) => <span>{`value:${value?.in?.join(",") ?? "none"}`}</span>}
      </Form.Field>
    </Form.Form>
  );
};
Fixture.displayName = "Fixture";

describe("Status.Filter.MenuItem", () => {
  it("should add the chosen variant to the where", async () => {
    await renderWithConsole(<Fixture />);
    fireEvent.click(await findDialogTrigger());
    const option = await waitFor(() => screen.getByText("Error"));
    fireEvent.click(option);
    await waitFor(() => expect(screen.getByText("value:error")).toBeTruthy());
  });

  it("should drop the variant line when the selection empties", async () => {
    await renderWithConsole(<Fixture />);
    fireEvent.click(await findDialogTrigger());
    fireEvent.click(await waitFor(() => screen.getByText("Error")));
    await waitFor(() => expect(screen.getByText("value:error")).toBeTruthy());
    fireEvent.click(screen.getByText("Error"));
    await waitFor(() => expect(screen.getByText("value:none")).toBeTruthy());
  });

  it("should keep a variant that the where selects with eq", async () => {
    await renderWithConsole(<Fixture variant={{ eq: "error" }} />);
    fireEvent.click(await findDialogTrigger());
    fireEvent.click(await waitFor(() => screen.getByText("Success")));
    await waitFor(() => expect(screen.getByText("value:error,success")).toBeTruthy());
  });
});
