// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label, ranger, view } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Form } from "@synnaxlabs/lyra/form";
import { deep, uuid } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Label } from "@/platform/label";
import { View } from "@/platform/view";
import {
  awaitGranted,
  createConsoleWrapper,
  renderSuspended,
  uniqueName,
} from "@/testutil";

const path = deep.path<z.infer<typeof schema>>();

const client = createTestClient();

const schema = z.object({ where: ranger.whereZ.optional() });

const Fixture = ({ labels }: { labels?: label.Key[] }): ReactElement => {
  const methods = Form.use({
    schema,
    values: { where: labels == null ? undefined : { labels } },
  });
  return (
    <View.Frame resourceType="range" icon="Label">
      <Form.Form<typeof schema> {...methods}>
        <Label.Filter.Chips />
        <Form.Field path={deep.present(path.where).labels} showLabel={false}>
          {({ value }) => <span>{`value:${value?.join(",") ?? "none"}`}</span>}
        </Form.Field>
      </Form.Form>
    </View.Frame>
  );
};
Fixture.displayName = "Fixture";

const createLabels = async (count: number): Promise<label.Label[]> =>
  await Promise.all(
    Array.from(
      { length: count },
      async (_, i) =>
        await client.labels.create({
          name: uniqueName(`label-${i}`),
          color: "#F00000",
        }),
    ),
  );

const renderFixture = async (labels?: label.Key[]): Promise<void> => {
  // Warm the subject's policy and role caches so the edit affordance's permission
  // check does not wait on the network mid-test.
  await awaitGranted(client, view.resourceID(uuid.create()), "update");
  const { wrapper } = await createConsoleWrapper({ client });
  await renderSuspended(<Fixture labels={labels} />, { wrapper });
};

describe("Label.Filter.Chips", () => {
  it("should name one label after has", async () => {
    const [only] = await createLabels(1);
    await renderFixture([only.key]);
    await waitFor(() => expect(screen.getByText(only.name)).toBeTruthy());
    expect(screen.getByText("Labels")).toBeTruthy();
    expect(screen.getByText("has")).toBeTruthy();
  });

  it("should name two labels after has any of", async () => {
    const [first, second] = await createLabels(2);
    await renderFixture([first.key, second.key]);
    await waitFor(() =>
      expect(screen.getByText(`${first.name}, ${second.name}`)).toBeTruthy(),
    );
    expect(screen.getByText("has any of")).toBeTruthy();
  });

  it("should count three or more labels", async () => {
    const labels = await createLabels(3);
    await renderFixture(labels.map(({ key }) => key));
    await waitFor(() => expect(screen.getByText("3 labels")).toBeTruthy());
  });

  it("should drop the labels line when the chip is removed", async () => {
    const [first, second] = await createLabels(2);
    await renderFixture([first.key, second.key]);
    fireEvent.click(
      await screen.findByRole("button", { name: "Remove Labels filter" }),
    );
    await waitFor(() => expect(screen.getByText("value:none")).toBeTruthy());
    expect(screen.queryByText("Labels")).toBeNull();
  });

  it("should render nothing when no labels are selected", async () => {
    const { wrapper } = await createConsoleWrapper({ client });
    await renderSuspended(<Fixture />, { wrapper });
    await waitFor(() => expect(screen.getByText("All ranges")).toBeTruthy());
    expect(screen.queryByText("Labels")).toBeNull();
  });
});
