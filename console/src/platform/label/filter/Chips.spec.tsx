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
import { render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Label } from "@/platform/label";
import { View } from "@/platform/view";
import { createConsoleWrapper } from "@/testutil";

const TIMEOUT = { timeout: 5000 };

const schema = z.object({
  query: z.object({ hasLabels: z.array(z.string()) }),
});

const Fixture = ({ hasLabels }: { hasLabels: label.Key[] }): ReactElement => {
  const methods = Form.use({ schema, values: { query: { hasLabels } } });
  return (
    <View.Frame resourceType="label" icon="Label">
      <Form.Form<typeof schema> {...methods}>
        <Label.Filter.Chips />
      </Form.Form>
    </View.Frame>
  );
};
Fixture.displayName = "Fixture";

describe("Label.Filter.Chips", () => {
  it("should render a tag for each selected label", async () => {
    const client = createTestClient();
    const name = id.create();
    const label = await client.labels.create({ name, color: "#F00000" });
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Fixture hasLabels={[label.key]} />, { wrapper });
    await waitFor(() => expect(screen.getByText(name)).toBeTruthy(), TIMEOUT);
    expect(screen.getByText("Labels")).toBeTruthy();
  });

  it("should render nothing when no labels are selected", async () => {
    const client = createTestClient();
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Fixture hasLabels={[]} />, { wrapper });
    await waitFor(() => expect(screen.getByText("All Labels")).toBeTruthy(), TIMEOUT);
    expect(screen.queryByText("Labels")).toBeNull();
  });
});
