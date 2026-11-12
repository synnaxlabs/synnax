// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger, type Synnax as Client } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { Flux } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Range } from "@/feature/range";
import { Modals } from "@/platform/modals";
import { createTestRange } from "@/platform/range/testutil";
import {
  awaitGranted,
  createConsoleWrapper,
  getIconButton,
  renderSuspended,
  selectTestProject,
} from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

interface HarnessProps {
  keys: ranger.Key[];
}

const Harness = ({ keys }: HarnessProps): ReactElement => {
  const { data, getItem, subscribe, more, answered } = Flux.useList((c) => c.ranges, {
    name: "ranges",
    query: { where: { key: { in: keys } } },
  });
  return (
    <Range.List.List
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      more={more}
      answered={answered}
    />
  );
};

const renderList = async (
  keys: ranger.Key[],
  as: Client = client,
): Promise<HTMLElement> => {
  const { wrapper, store } = await createConsoleWrapper({ client: as });
  await selectTestProject(store, client);
  const { container } = await renderSuspended(
    <>
      <Harness keys={keys} />
      <Modals.Stack />
    </>,
    { wrapper },
  );
  return container;
};

interface ShowRangeParams {
  as?: Client;
  granted?: boolean;
}

const showRange = async ({
  as = client,
  granted = true,
}: ShowRangeParams = {}): Promise<[HTMLElement, ranger.Range]> => {
  const rng = await createTestRange(client);
  // The item draws its name again once the update grant lands, so a granted subject
  // waits for the grant before the spec reads the item.
  if (granted) await awaitGranted(as, ranger.resourceID(rng.key), "update");
  const container = await renderList([rng.key], as);
  await within(container).findByText(rng.name);
  return [container, rng];
};

describe("range/list/List", () => {
  it("offers Reload Console from a range's context menu", async () => {
    const [, rng] = await showRange();
    fireEvent.contextMenu(await screen.findByText(rng.name));
    expect(await screen.findByText("Reload Console")).toBeTruthy();
  });

  it("should open the Sift upload modal from a range's context menu", async () => {
    const [, rng] = await showRange();
    fireEvent.contextMenu(await screen.findByText(rng.name));
    fireEvent.click(await screen.findByText("Upload to Sift"));
    expect(await screen.findByText(`Upload ${rng.name} to Sift`)).toBeTruthy();
  });

  it("should end a range on the Core from its stage selector", async () => {
    const [container, rng] = await showRange();
    const item = within(container).getByLabelText(rng.name);
    fireEvent.click(getIconButton(item, "in-progress"));
    fireEvent.click(await screen.findByText("Complete"));
    await waitFor(async () => {
      const { timeRange } = await client.ranges.retrieve(rng.key);
      expect(timeRange.end.before(rng.timeRange.end)).toBe(true);
    });
  });
});

describe("range/list/List permissions", () => {
  const showEmptyState = async (as: Client): Promise<HTMLElement> => {
    const container = await renderList([], as);
    await within(container).findByText("No ranges found", undefined, {
      timeout: 5000,
    });
    return container;
  };

  it("should freeze the stage selector for a viewer", async () => {
    const [container] = await showRange({
      as: await roles.get("Viewer"),
      granted: false,
    });
    expect(getIconButton(container, "in-progress").getAttribute("aria-disabled")).toBe(
      "true",
    );
  });

  it("should leave the stage selector live for an operator", async () => {
    const [container] = await showRange({ as: await roles.get("Operator") });
    expect(
      getIconButton(container, "in-progress").getAttribute("aria-disabled"),
    ).toBeNull();
  });

  it("should withhold the create action from a viewer", async () => {
    const container = await showEmptyState(await roles.get("Viewer"));
    expect(within(container).queryByText("Create range")).toBeNull();
  });

  it("should offer the create action to an operator", async () => {
    const container = await showEmptyState(await roles.get("Operator"));
    expect(await within(container).findByText("Create range")).toBeTruthy();
  });
});
