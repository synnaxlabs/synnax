// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import {
  act,
  fireEvent,
  renderHook,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Vcs } from "@/feature/vcs";
import { Notifications } from "@/platform/notifications";
import { Vcs as PlatformVcs } from "@/platform/vcs";
import {
  assertDefined,
  createConsoleWrapper,
  installPortalRoot,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

/** Creates Arcs named `${prefix}_a`, `${prefix}_b`, and so on. */
const createArcs = async (prefix: string, count: number): Promise<arc.Arc[]> =>
  await client.arcs.create(
    Array.from({ length: count }, (_, i): arc.New => ({
      name: `${prefix}_${String.fromCharCode(97 + i)}`,
      type: "automation",
      source: { mode: "graph", graph: { nodes: [], edges: [] } },
    })),
  );

/**
 * Renders the toolbar and its notifications.
 * @returns the function that opens a plan in the toolbar.
 */
const renderToolbar = async () => {
  const { wrapper: Console } = await createConsoleWrapper({ client });
  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Vcs.Provider>
        {children}
        {Vcs.TOOLBAR.content}
        <Notifications.Feed notifications={Vcs.NOTIFICATIONS} />
      </Vcs.Provider>
    </Console>
  );
  const { result } = renderHook(() => PlatformVcs.useOpenPlan(), { wrapper });
  return result;
};

const findToast = async (message: string): Promise<HTMLElement> =>
  await waitFor(() => {
    const toast = screen
      .getAllByRole("status")
      .find((el) => within(el).queryByText(message) != null);
    assertDefined(toast, `no notification says ${message}`);
    return toast;
  });

describe("Vcs.TOOLBAR", () => {
  installPortalRoot();

  // A fresh Core has no version control yet.
  beforeAll(async () => {
    const { enabled } = await client.vcs.status();
    if (!enabled)
      await client.vcs.connect({ configuration: "main", remoteUrl: "", token: "" });
  });

  describe("discard", () => {
    it("should report the discarded change with an Undo action", async () => {
      const [discarded] = await createArcs(uniqueName("discard"), 1);
      await renderToolbar();
      fireEvent.click(
        await screen.findByLabelText(`Discard changes to ${discarded.name}`),
      );
      const toast = await findToast("Discarded changes");
      expect(within(toast).getByText(discarded.name)).toBeTruthy();
      expect(within(toast).getByRole("button", { name: "Undo" })).toBeTruthy();
    });
  });

  describe("apply", () => {
    interface Spec {
      count: number;
      apply: string;
      message: string;
      description: (prefix: string) => string;
    }

    const SPECS: Spec[] = [
      {
        count: 1,
        apply: "Apply 1 change",
        message: "Restored 1 change",
        description: (p) => `Updated ${p}_a.`,
      },
      {
        count: 2,
        apply: "Apply 2 changes",
        message: "Restored 2 changes",
        description: (p) => `Updated ${p}_a and ${p}_b.`,
      },
      {
        count: 3,
        apply: "Apply 3 changes",
        message: "Restored 3 changes",
        description: (p) => `Updated ${p}_a, ${p}_b, and ${p}_c.`,
      },
      {
        count: 4,
        apply: "Apply 4 changes",
        message: "Restored 4 changes",
        description: (p) => `Updated ${p}_a, ${p}_b, ${p}_c, and 1 more.`,
      },
    ];

    it.each(SPECS)(
      "should report '$message' with the restored names and an Undo action",
      async ({ count, apply, message, description }) => {
        const prefix = uniqueName("restore");
        const arcs = await createArcs(prefix, count);
        const { head, configuration } = await client.vcs.status();
        assertDefined(head);
        const openPlan = await renderToolbar();
        // Restoring Arcs that the last commit does not hold deletes them.
        act(() =>
          openPlan.current({
            target: {
              kind: "restore",
              commit: head.hash,
              resources: arcs.map(({ key }) =>
                PlatformVcs.idString(arc.resourceID(key)),
              ),
            },
            title: `Restore ${head.shortHash}`,
            after: head.hash,
            afterLabel: head.shortHash,
          }),
        );
        fireEvent.click(await screen.findByText(apply));
        const toast = await findToast(message);
        expect(within(toast).getByText(configuration)).toBeTruthy();
        expect(within(toast).getByText(description(prefix))).toBeTruthy();
        expect(within(toast).getByRole("button", { name: "Undo" })).toBeTruthy();
      },
    );

    it("should leave out the Undo action when the apply changes nothing", async () => {
      const { head } = await client.vcs.status();
      assertDefined(head);
      const openPlan = await renderToolbar();
      act(() =>
        openPlan.current({
          target: { kind: "pull" },
          title: "Pull 0 commits",
          after: head.hash,
          afterLabel: "Incoming",
        }),
      );
      await screen.findByText("Nothing changes on the Core");
      fireEvent.click(screen.getByText("Pull"));
      const toast = await findToast("Pulled with no changes");
      expect(within(toast).queryByRole("button", { name: "Undo" })).toBeNull();
    });
  });
});
