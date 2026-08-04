// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Form } from "@synnaxlabs/pluto";
import { render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Label } from "@/platform/label";
import { searchAndClickLabel } from "@/platform/label/testutil";
import { createConsoleWrapper, uniqueName } from "@/testutil";

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
          return <span>{`value:${value.join(",")}`}</span>;
        }}
      </Form.Field>
    </Form.Form>
  );
};
Fixture.displayName = "Fixture";

describe("Label.Filter.MenuItem", () => {
  it("should add a selected label's key to the query field", async () => {
    const client = createTestClient();
    const name = uniqueName("label");
    const label = await client.labels.create({ name, color: "#000000" });
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Fixture />, { wrapper });
    await searchAndClickLabel(name);
    await waitFor(() => expect(screen.getByText(`value:${label.key}`)).toBeTruthy());
  });
});
