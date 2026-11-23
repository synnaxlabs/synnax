// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { rack } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Rack } from "@synnaxlabs/pluto";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { act, fireEvent, renderHook, screen, waitFor } from "@testing-library/react";
import type { PropsWithChildren, ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Modals } from "@/platform/modals";
import { findButton } from "@/platform/modals/testutil";
import { Tree } from "@/platform/tree";
import {
  createBaseProps,
  createResource,
  createSelection,
  createState,
} from "@/platform/tree/testutil";
import { createConsoleWrapper, createTestStore } from "@/testutil";

const client = createTestClient();

const useDelete = Tree.createUseDelete({
  type: "Rack",
  useUpdate: Rack.useDelete,
  convertKey: Number,
});

const createRack = async () => await client.racks.create({ name: uniqueName("rack") });

const setup = async (target: rack.Rack) => {
  const store = await createTestStore();
  const propsFor = (r: rack.Rack): Tree.ContextMenuProps => {
    const id = rack.resourceID(r.key);
    return {
      ...createBaseProps({ client, store }),
      selection: createSelection({ ids: [id] }),
      state: createState([createResource(id, r.name)]),
    };
  };
  const { wrapper: Console } = await createConsoleWrapper({ client, store });
  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      {children}
      <Modals.Stack />
    </Console>
  );
  const { result, rerender } = renderHook(
    (props: Tree.ContextMenuProps) => useDelete(props),
    { wrapper, initialProps: propsFor(target) },
  );
  return {
    /** Invokes the delete, opening its confirmation modal. */
    del: async () => {
      // The confirmation content suspends, and content that suspends inside a
      // synchronous act scope is discarded, so the invocation needs an awaited one.
      await act(async () => {
        result.current();
      });
    },
    /** Points the hook at another resource, as a changed tree selection would. */
    select: (r: rack.Rack) => rerender(propsFor(r)),
  };
};

const rackExists = async (key: rack.Key): Promise<boolean> => {
  try {
    await client.racks.retrieve({ key });
    return true;
  } catch {
    return false;
  }
};

const confirmationFor = (r: rack.Rack): string =>
  `Are you sure you want to delete ${r.name}?`;

describe("createUseDelete", () => {
  it("should delete the selected resource after the user confirms", async () => {
    const r = await createRack();
    const { del } = await setup(r);
    await del();
    await waitFor(() => expect(screen.getByText(confirmationFor(r))).toBeTruthy());
    fireEvent.click(findButton("Delete"));
    await waitFor(async () => expect(await rackExists(r.key)).toBe(false));
  });

  it("should leave the resource in place when the user cancels", async () => {
    const r = await createRack();
    const control = await createRack();
    const { del, select } = await setup(r);
    await del();
    await waitFor(() => expect(screen.getByText(confirmationFor(r))).toBeTruthy());
    fireEvent.click(findButton("Cancel"));
    await waitFor(() => expect(screen.queryByText(confirmationFor(r))).toBeNull());
    // A confirmed deletion through the same mutation path settles any erroneous
    // in-flight delete of the canceled target before the absence assert.
    select(control);
    await del();
    await waitFor(() =>
      expect(screen.getByText(confirmationFor(control))).toBeTruthy(),
    );
    fireEvent.click(findButton("Delete"));
    await waitFor(async () => expect(await rackExists(control.key)).toBe(false));
    expect(await rackExists(r.key)).toBe(true);
  });
});
