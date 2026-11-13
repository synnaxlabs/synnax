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
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Label } from "@/platform/label";
import { View } from "@/platform/view";
import { enableEditing, openFilterProperty } from "@/platform/view/testutil";
import {
  awaitGranted,
  createConsoleWrapper,
  renderSuspended,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const path = deep.path<z.infer<typeof schema>>();

const schema = z.object({ where: ranger.whereZ.optional() });

const RANGES: View.Filter.Properties<ranger.Where> = {
  labels: Label.Filter.createProperty({ unlabeled: true }),
};

const NO_UNLABELED: View.Filter.Properties<ranger.Where> = {
  labels: Label.Filter.createProperty(),
};

interface FixtureProps {
  where?: ranger.Where;
  properties: View.Filter.Properties<ranger.Where>;
}

const Fixture = ({ where, properties }: FixtureProps): ReactElement => {
  const methods = Form.use({ schema, values: { where } });
  return (
    <View.Frame resourceType="range" icon="Label">
      <Form.Form<typeof schema> {...methods}>
        <View.Filter.Menu properties={properties} />
        <View.Filter.Chips properties={properties} />
        <Form.Field path={path.where} showLabel={false}>
          {({ value }) => <span>{`where:${JSON.stringify(value) ?? "none"}`}</span>}
        </Form.Field>
      </Form.Form>
    </View.Frame>
  );
};
Fixture.displayName = "Fixture";

const renderFixture = async (
  where?: ranger.Where,
  properties: View.Filter.Properties<ranger.Where> = RANGES,
): Promise<void> => {
  // Warm the policy caches so the edit toggle does not wait on the network mid-test.
  await awaitGranted(client, view.resourceID(uuid.create()), "update");
  const { wrapper } = await createConsoleWrapper({ client });
  await renderSuspended(<Fixture where={where} properties={properties} />, {
    wrapper,
  });
};

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

const expectWhere = async (where: ranger.Where | undefined): Promise<void> => {
  await waitFor(() =>
    expect(screen.getByText(`where:${JSON.stringify(where) ?? "none"}`)).toBeTruthy(),
  );
};

const pickMatch = async (current: string, next: string): Promise<void> => {
  fireEvent.click(await screen.findByRole("button", { name: current }));
  const dialog = await screen.findByRole("dialog", { name: "Match" });
  fireEvent.click(within(dialog).getByText(next));
};

describe("Label.Filter.createProperty", () => {
  describe("chip", () => {
    it("should name one label after has", async () => {
      const [only] = await createLabels(1);
      await renderFixture({ labels: [only.key] });
      expect(await screen.findByText(only.name)).toBeTruthy();
      expect(screen.getByText("has")).toBeTruthy();
    });

    it("should name two labels after has any of", async () => {
      const [first, second] = await createLabels(2);
      await renderFixture({ labels: [first.key, second.key] });
      expect(await screen.findByText(`${first.name}, ${second.name}`)).toBeTruthy();
      expect(screen.getByText("has any of")).toBeTruthy();
    });

    it("should count three or more labels", async () => {
      const labels = await createLabels(3);
      await renderFixture({ labels: labels.map(({ key }) => key) });
      expect(await screen.findByText("3 labels")).toBeTruthy();
    });

    it("should name only the labels that still exist", async () => {
      const [kept, deleted] = await createLabels(2);
      await client.labels.delete(deleted.key);
      await renderFixture({ labels: [kept.key, deleted.key] });
      expect(await screen.findByText(kept.name)).toBeTruthy();
      expect(screen.getByText("has")).toBeTruthy();
    });

    it("should count no labels when every label is deleted", async () => {
      const [deleted] = await createLabels(1);
      await client.labels.delete(deleted.key);
      await renderFixture({ labels: [deleted.key] });
      expect(await screen.findByText("0 labels")).toBeTruthy();
      expect(screen.getByText("has any of")).toBeTruthy();
    });

    it("should read labels that a range must all have", async () => {
      const [first, second] = await createLabels(2);
      await renderFixture({
        and: [{ and: [{ labels: [first.key] }, { labels: [second.key] }] }],
      });
      expect(await screen.findByText("has all of")).toBeTruthy();
    });

    it("should read labels that a range must not have", async () => {
      const [only] = await createLabels(1);
      await renderFixture({ not: { labels: [only.key] } });
      expect(await screen.findByText("has none of")).toBeTruthy();
    });

    it("should read ranges with no labels", async () => {
      await renderFixture({ labeled: false });
      expect(await screen.findByText("No labels")).toBeTruthy();
      expect(screen.getByText("is")).toBeTruthy();
    });

    it("should require all labels once the match is all", async () => {
      const [first, second] = await createLabels(2);
      await renderFixture({ labels: [first.key, second.key] });
      await enableEditing();
      await pickMatch("has any of", "has all of");
      await expectWhere({
        and: [{ and: [{ labels: [first.key] }, { labels: [second.key] }] }],
      });
    });

    it("should exclude the labels once the match is none", async () => {
      const [first, second] = await createLabels(2);
      await renderFixture({ labels: [first.key, second.key] });
      await enableEditing();
      await pickMatch("has any of", "has none of");
      await expectWhere({ and: [{ not: { labels: [first.key, second.key] } }] });
    });

    it("should open the picker of the menu below the chip", async () => {
      const [first, second] = await createLabels(2);
      await renderFixture({ labels: [first.key] });
      await enableEditing();
      fireEvent.click(await screen.findByRole("button", { name: first.name }));
      const picker = await screen.findByRole("dialog", { name: "Labels" });
      fireEvent.change(within(picker).getByPlaceholderText("Search labels..."), {
        target: { value: second.name },
      });
      fireEvent.click(await within(picker).findByText(second.name));
      await expectWhere({ and: [{ labels: [first.key, second.key] }] });
    });
  });

  describe("menu", () => {
    it("should add a picked label to the filter", async () => {
      const [only] = await createLabels(1);
      await renderFixture();
      await enableEditing();
      const picker = await openFilterProperty("Labels");
      fireEvent.change(within(picker).getByPlaceholderText("Search labels..."), {
        target: { value: only.name },
      });
      fireEvent.click(await within(picker).findByText(only.name));
      await expectWhere({ and: [{ labels: [only.key] }] });
    });

    it("should drop the condition once no label is picked", async () => {
      const [only] = await createLabels(1);
      await renderFixture({ labels: [only.key] });
      await enableEditing();
      const picker = await openFilterProperty("Labels");
      fireEvent.change(within(picker).getByPlaceholderText("Search labels..."), {
        target: { value: only.name },
      });
      fireEvent.click(await within(picker).findByText(only.name));
      await expectWhere(undefined);
    });

    it("should pick ranges with no labels", async () => {
      await renderFixture();
      await enableEditing();
      const picker = await openFilterProperty("Labels");
      fireEvent.click(within(picker).getByText("No labels"));
      await expectWhere({ and: [{ labeled: false }] });
    });

    it("should not offer no labels where the filter cannot read it", async () => {
      await renderFixture(undefined, NO_UNLABELED);
      await enableEditing();
      const picker = await openFilterProperty("Labels");
      await within(picker).findByPlaceholderText("Search labels...");
      expect(within(picker).queryByText("No labels")).toBeNull();
    });
  });
});
