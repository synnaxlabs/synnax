// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status, type Synnax as Client, view } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Form } from "@synnaxlabs/lyra/form";
import { deep, filter, uuid } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Label } from "@/platform/label";
import { Status } from "@/platform/status";
import { View } from "@/platform/view";
import { openFilterProperty } from "@/platform/view/testutil";
import {
  createConsoleWrapper,
  createTestClientWithGrants,
  renderSuspended,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const path = deep.path<z.infer<typeof schema>>();

const schema = z.object({ where: status.whereZ.optional() });

const PROPERTIES: View.Filter.Properties<status.Where> = {
  labels: Label.Filter.createProperty(),
  variant: Status.Filter.VARIANT,
};

const Fixture = ({ where }: { where?: status.Where }): ReactElement => {
  const methods = Form.use({ schema, values: { where } });
  return (
    <View.Frame resourceType="status" icon="Status">
      <Form.Form<typeof schema> {...methods}>
        <View.Filter.Menu properties={PROPERTIES} />
        <View.Filter.Chips properties={PROPERTIES} />
        <Form.Field path={path.where} showLabel={false}>
          {({ value }) => <span>{`where:${JSON.stringify(value) ?? "none"}`}</span>}
        </Form.Field>
      </Form.Form>
    </View.Frame>
  );
};
Fixture.displayName = "Fixture";

const renderFixture = async (
  where?: status.Where,
  as: Client = client,
): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client: as });
  await renderSuspended(<Fixture where={where} />, { wrapper });
};

const expectWhere = async (where: status.Where | undefined): Promise<void> => {
  await waitFor(() =>
    expect(screen.getByText(`where:${JSON.stringify(where) ?? "none"}`)).toBeTruthy(),
  );
};

const pickVariant = async (name: string): Promise<void> => {
  const picker = await openFilterProperty("Variants");
  fireEvent.click(within(picker).getByText(name));
};

describe("View.Filter", () => {
  it("should hold each condition in its own node of the filter", async () => {
    await renderFixture();
    await pickVariant("Error");
    await expectWhere({ and: [{ variant: { in: ["error"] } }] });
  });

  it("should keep one condition for each property, in its place", async () => {
    const key = uuid.create();
    await renderFixture({
      and: [{ variant: { in: ["error"] } }, { key: { in: [key] } }],
    });
    await pickVariant("Warning");
    await expectWhere({
      and: [{ variant: { in: ["error", "warning"] } }, { key: { in: [key] } }],
    });
  });

  describe("two conditions on one property", () => {
    const TWO: status.Where = {
      and: [{ variant: { in: ["error"] } }, { variant: { in: ["info"] } }],
    };

    it("should edit only the condition of the chip", async () => {
      await renderFixture(TWO);
      await enableEditing();
      fireEvent.click(await screen.findByRole("button", { name: "Info" }));
      const picker = await screen.findByRole("dialog", { name: "Variants" });
      fireEvent.click(within(picker).getByText("Warning"));
      await expectWhere({
        and: [{ variant: { in: ["error"] } }, { variant: { in: ["info", "warning"] } }],
      });
    });

    it("should remove only the condition of the chip", async () => {
      await renderFixture(TWO);
      await enableEditing();
      const removes = await screen.findAllByRole("button", {
        name: "Remove Variants filter",
      });
      fireEvent.click(removes[1]);
      await expectWhere({ and: [{ variant: { in: ["error"] } }] });
    });
  });

  it("should show a chip for each field of one node", async () => {
    await renderFixture({ variant: { in: ["error"] }, key: { in: [uuid.create()] } });
    expect(await screen.findByText("Variants")).toBeTruthy();
    expect(screen.getByText("Error")).toBeTruthy();
  });

  it("should show a node that no property reads as text", async () => {
    const node: status.Where = { key: { in: [uuid.create()] } };
    await renderFixture(node);
    expect(await screen.findByText(filter.describe(node))).toBeTruthy();
  });

  it("should drop a node that no property reads with its chip", async () => {
    await renderFixture({
      and: [{ variant: { in: ["error"] } }, { key: { in: [uuid.create()] } }],
    });
    fireEvent.click(
      await screen.findByRole("button", { name: "Remove Filter filter" }),
    );
    await expectWhere({ and: [{ variant: { in: ["error"] } }] });
  });

  it("should drop the filter with the last chip", async () => {
    await renderFixture({ variant: { in: ["error"] } });
    fireEvent.click(
      await screen.findByRole("button", { name: "Remove Variants filter" }),
    );
    await expectWhere(undefined);
    expect(screen.queryByText("Variants")).toBeNull();
  });

  it("should clear two or more chips at once", async () => {
    await renderFixture({
      and: [{ variant: { in: ["error"] } }, { key: { in: [uuid.create()] } }],
    });
    fireEvent.click(await screen.findByRole("button", { name: "Clear" }));
    await expectWhere(undefined);
  });

  it("should not offer to clear a single chip", async () => {
    await renderFixture({ variant: { in: ["error"] } });
    await screen.findByRole("button", { name: "Remove Variants filter" });
    expect(screen.queryByRole("button", { name: "Clear" })).toBeNull();
  });

  it("should open the picker of a property beside the filter menu", async () => {
    await renderFixture();
    await openFilterProperty("Variants");
    expect(
      screen.getByRole("menuitem", { name: "Variants", expanded: true }),
    ).toBeTruthy();
  });

  it("should withhold the filter menu and the remove button from a reader of a saved view", async () => {
    const saved = await client.views.create({
      name: uniqueName("view"),
      type: "status",
    });
    const reader = await createTestClientWithGrants(client, {
      retrieve: [status.TYPE_RESOURCE_ID, view.TYPE_RESOURCE_ID],
    });
    await renderFixture({ variant: { in: ["error"] } }, reader);
    // A static view saves nothing, so a reader may change its filter.
    await screen.findByRole("button", { name: "Remove Variants filter" });
    expect(screen.getByRole("button", { name: "Filter" })).toBeTruthy();
    fireEvent.click(await screen.findByRole("tab", { name: saved.name }));
    await waitFor(() =>
      expect(
        screen.queryByRole("button", { name: "Remove Variants filter" }),
      ).toBeNull(),
    );
    expect(screen.queryByRole("button", { name: "Filter" })).toBeNull();
    expect(screen.getByText("Error")).toBeTruthy();
  });
});
