// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger, type Synnax as Client } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { Ranger } from "@synnaxlabs/pluto";
import { fireEvent, screen, within } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Range } from "@/feature/range";
import { Modals } from "@/platform/modals";
import { createChildTestRange, createTestRange } from "@/platform/range/testutil";
import { createConsoleWrapper, getIconButton, renderSuspended } from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

interface HarnessProps {
  parent: ranger.Key;
}

const Harness = ({ parent }: HarnessProps): ReactElement => {
  const { data, getItem, subscribe, retrieve } = Ranger.useListChildren({
    initialQuery: { key: parent },
  });
  return (
    <Range.List.List
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      retrieve={retrieve}
    />
  );
};

// The shared Core accumulates ranges across runs, so the list is scoped to the
// children of a fresh parent that only the test fills.
const renderList = async (
  parent: ranger.Range,
  as: Client = client,
): Promise<HTMLElement> => {
  const { wrapper } = await createConsoleWrapper({ client: as });
  const { container } = await renderSuspended(
    <>
      <Harness parent={parent.key} />
      <Modals.Stack />
    </>,
    { wrapper },
  );
  return container;
};

describe("range/list/List", () => {
  it("lists the children of the queried range", async () => {
    const parent = await createTestRange(client);
    const child = await createChildTestRange(client, parent);
    const other = await createTestRange(client);
    const container = await renderList(parent);
    expect(await within(container).findByText(child.name)).toBeTruthy();
    expect(within(container).queryByText(other.name)).toBeNull();
  });

  it("offers Reload Console from a range's context menu", async () => {
    const parent = await createTestRange(client);
    const child = await createChildTestRange(client, parent);
    const container = await renderList(parent);
    fireEvent.contextMenu(await within(container).findByText(child.name));
    expect(await screen.findByText("Reload Console")).toBeTruthy();
  });
});

describe("range/list/List permissions", () => {
  it("should freeze the stage selector for a viewer", async () => {
    const parent = await createTestRange(client);
    const child = await createChildTestRange(client, parent);
    const container = await renderList(parent, await roles.get("Viewer"));
    await within(container).findByText(child.name);
    expect(getIconButton(container, "in-progress").getAttribute("aria-disabled")).toBe(
      "true",
    );
  });

  it("should leave the stage selector live for an operator", async () => {
    const parent = await createTestRange(client);
    const child = await createChildTestRange(client, parent);
    const container = await renderList(parent, await roles.get("Operator"));
    await within(container).findByText(child.name);
    expect(
      getIconButton(container, "in-progress").getAttribute("aria-disabled"),
    ).toBeNull();
  });
});
