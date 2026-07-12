// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Docs } from "@/platform/docs";
import { Session } from "@/session";
import {
  createConsoleWrapper,
  renderWithConsole,
  resolveFocusedTab,
  uniqueName,
} from "@/testutil";

const getIframe = (container: HTMLElement): HTMLIFrameElement => {
  const iframe = container.querySelector("iframe");
  if (iframe == null) throw new Error("iframe not found");
  return iframe;
};

describe("Docs", () => {
  describe("Docs renderer", () => {
    it("should point the iframe at the docs host with the default path", async () => {
      const { container } = await renderWithConsole(<Docs.Docs />);
      await waitFor(() => expect(container.querySelector("iframe")).not.toBeNull());
      const src = getIframe(container).src;
      expect(src).toContain("docs.synnaxlabs.com");
      expect(src).toContain("reference/console/get-started");
    });

    it("should use the persisted location path when one is set", async () => {
      const { container } = await renderWithConsole(<Docs.Docs />, {
        preloadedState: {
          [Session.Docs.SLICE_NAME]: {
            version: 0,
            location: { path: "reference/cluster/quick-start", heading: "" },
          },
        },
      });
      await waitFor(() => expect(container.querySelector("iframe")).not.toBeNull());
      expect(getIframe(container).src).toContain("reference/cluster/quick-start");
    });

    it("should encode the active theme into the frame query string", async () => {
      const { container } = await renderWithConsole(<Docs.Docs />);
      await waitFor(() => expect(container.querySelector("iframe")).not.toBeNull());
      const src = getIframe(container).src;
      expect(src).toMatch(/theme=(dark|light)/);
      expect(src).toContain("noHeader=true");
    });

    it("should show the loading watermark until the iframe finishes loading", async () => {
      const { container } = await renderWithConsole(<Docs.Docs />);
      await waitFor(() => expect(container.querySelector("iframe")).not.toBeNull());
      expect(container.querySelector("svg")).not.toBeNull();
      act(() => {
        fireEvent.load(getIframe(container));
      });
      await waitFor(() => expect(container.querySelector("svg")).toBeNull());
    });

    it("should dispatch a location update when the frame posts a message", async () => {
      const { container, store } = await renderWithConsole(<Docs.Docs />);
      await waitFor(() => expect(container.querySelector("iframe")).not.toBeNull());
      act(() => {
        window.dispatchEvent(
          new MessageEvent("message", {
            data: { path: "reference/console/schematics", heading: "creating" },
          }),
        );
      });
      await waitFor(() =>
        expect(Session.Docs.selectLocation(store.getState())).toEqual({
          path: "reference/console/schematics",
          heading: "creating",
        }),
      );
    });
  });

  describe("OpenButton", () => {
    it("should open the documentation view as a tab when clicked", async () => {
      const client = createTestClient();
      const proj = await client.projects.create({
        name: uniqueName("proj"),
        layout: {},
      });
      const { wrapper, store } = await createConsoleWrapper({ client });
      render(<Docs.OpenButton />, { wrapper });
      store.dispatch(Session.Project.select(proj.key));
      fireEvent.click(screen.getByRole("button"));
      const tab = await resolveFocusedTab(store, client);
      if (tab.variant !== "view") throw new Error("expected a view tab");
      expect(tab.type).toBe(Docs.TAB_TYPE);
    });
  });
});
