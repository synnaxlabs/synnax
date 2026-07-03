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
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Label } from "@/platform/label";
import { View } from "@/platform/view";
import { enableEditing } from "@/platform/view/testutil";
import { createConsoleWrapper, findTagCloseButton, uniqueName } from "@/testutil";

const client = createTestClient();

const schema = z.object({
  query: z.object({ hasLabels: z.array(z.string()) }),
});

const Fixture = ({ hasLabels }: { hasLabels: label.Key[] }): ReactElement => {
  const methods = Form.use({ schema, values: { query: { hasLabels } } });
  return (
    <View.Frame resourceType="label" icon="Label">
      <Form.Form<typeof schema> {...methods}>
        <Label.Filter.Chips />
        <Form.Field<label.Key[]> path="query.hasLabels" showLabel={false}>
          {({ value }) => <span>{`value:${value.join(",")}`}</span>}
        </Form.Field>
      </Form.Form>
    </View.Frame>
  );
};
Fixture.displayName = "Fixture";

describe("Label.Filter.Chips", () => {
  it("should remove the label's key from the filter query when its chip is closed", async () => {
    const kept = await client.labels.create({
      name: uniqueName("kept"),
      color: "#F00000",
    });
    const removed = await client.labels.create({
      name: uniqueName("removed"),
      color: "#00F000",
    });
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Fixture hasLabels={[kept.key, removed.key]} />, { wrapper });
    await waitFor(() => expect(screen.getByText(removed.name)).toBeTruthy());
    await enableEditing();
    const close = await waitFor(() => findTagCloseButton(removed.name));
    fireEvent.click(close);
    await waitFor(() => expect(screen.getByText(`value:${kept.key}`)).toBeTruthy());
    await waitFor(() => expect(screen.queryByText(removed.name)).toBeNull());
    expect(screen.getByText(kept.name)).toBeTruthy();
  });

  it("should render nothing when no labels are selected", async () => {
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Fixture hasLabels={[]} />, { wrapper });
    await waitFor(() => expect(screen.getByText("All Labels")).toBeTruthy());
    expect(screen.queryByText("Labels")).toBeNull();
  });
});
