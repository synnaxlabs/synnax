// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status, view } from "@synnaxlabs/client";
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
import { enableEditing, openFilterProperty } from "@/platform/view/testutil";
import { awaitGranted, createConsoleWrapper, renderSuspended } from "@/testutil";

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

const renderFixture = async (where?: status.Where): Promise<void> => {
  // Warm the policy caches so the edit toggle does not wait on the network mid-test.
  await awaitGranted(client, view.resourceID(uuid.create()), "update");
  const { wrapper } = await createConsoleWrapper({ client });
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
    await enableEditing();
    await pickVariant("Error");
    await expectWhere({ and: [{ variant: { in: ["error"] } }] });
  });

  it("should keep one condition for each property, in its place", async () => {
    const key = uuid.create();
    await renderFixture({
      and: [{ variant: { in: ["error"] } }, { key: { in: [key] } }],
    });
    await enableEditing();
    await pickVariant("Warning");
    await expectWhere({
      and: [{ variant: { in: ["error", "warning"] } }, { key: { in: [key] } }],
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
    await enableEditing();
    fireEvent.click(
      await screen.findByRole("button", { name: "Remove Filter filter" }),
    );
    await expectWhere({ and: [{ variant: { in: ["error"] } }] });
  });

  it("should drop the filter with the last chip", async () => {
    await renderFixture({ variant: { in: ["error"] } });
    await enableEditing();
    fireEvent.click(
      await screen.findByRole("button", { name: "Remove Variants filter" }),
    );
    await expectWhere(undefined);
    expect(screen.queryByText("Variants")).toBeNull();
  });

  it("should withhold the remove button until the view is editable", async () => {
    await renderFixture({ variant: { in: ["error"] } });
    expect(await screen.findByText("Error")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Remove Variants filter" })).toBeNull();
    await enableEditing();
    expect(
      await screen.findByRole("button", { name: "Remove Variants filter" }),
    ).toBeTruthy();
  });

  it("should clear two or more chips at once", async () => {
    await renderFixture({
      and: [{ variant: { in: ["error"] } }, { key: { in: [uuid.create()] } }],
    });
    expect(await screen.findByText("Error")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Clear" })).toBeNull();
    await enableEditing();
    fireEvent.click(await screen.findByRole("button", { name: "Clear" }));
    await expectWhere(undefined);
  });

  it("should not offer to clear a single chip", async () => {
    await renderFixture({ variant: { in: ["error"] } });
    await enableEditing();
    await screen.findByRole("button", { name: "Remove Variants filter" });
    expect(screen.queryByRole("button", { name: "Clear" })).toBeNull();
  });

  it("should open the picker of a property beside the filter menu", async () => {
    await renderFixture();
    await enableEditing();
    await openFilterProperty("Variants");
    expect(
      screen.getByRole("menuitem", { name: "Variants", expanded: true }),
    ).toBeTruthy();
  });

  it("should hide the filter menu while the view is not editable", async () => {
    await renderFixture();
    expect(await screen.findByText("where:none")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Filter" })).toBeNull();
  });
});
