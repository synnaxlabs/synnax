// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { auth } from "@synnaxlabs/client";
import { fireEvent, render, screen } from "@testing-library/react";
import { assert, describe, expect, it, onTestFinished, vi } from "vitest";

// A serving Core skips the Core step, and detection reports one for every browser
// build. The module is the only seam.
vi.mock("@/platform/core/detectConnection", () => ({
  detectConnection: () => null,
}));

import { Auth } from "@/feature/auth";
import { Session } from "@/session";
import { createCore } from "@/session/core/testutil";
import { createSessionConsoleWrapper } from "@/testutil";

// A dead port: only the methods request is answered, by the stub below.
const CORE = createCore("alpha", { name: "Alpha", port: 9098 });

const PROVIDERS: auth.OIDCLogin[] = Array.from({ length: 10 }, (_, i) => ({
  provider: `idp-${i}`,
  issuer: `https://idp-${i}.example.com`,
  authorizationEndpoint: `https://idp-${i}.example.com/authorize`,
  clientId: "synnax",
}));

const serveProviders = (): void => {
  const real = globalThis.fetch;
  const spy = vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
    const url = input instanceof Request ? input.url : input.toString();
    if (!url.endsWith("/auth/methods")) return await real(input, init);
    const body = JSON.stringify({
      methods: ["password", "oidc"],
      providers: PROVIDERS,
    });
    return new Response(body, { headers: { "Content-Type": "application/json" } });
  });
  onTestFinished(() => spy.mockRestore());
};

const scrollParent = (el: HTMLElement): HTMLElement => {
  for (let p = el.parentElement; p != null; p = p.parentElement)
    if (getComputedStyle(p).overflowY === "auto") return p;
  throw new Error("no scrolling ancestor");
};

const buttonOf = (text: string): HTMLElement => {
  const button = screen.getByText(text).closest("button");
  assert(button != null);
  return button;
};

describe("auth/Login", () => {
  it("should scroll many providers in a window of three whole buttons", async () => {
    serveProviders();
    const { wrapper } = await createSessionConsoleWrapper({
      client: null,
      preloadedState: {
        [Session.Core.SLICE_NAME]: {
          ...Session.Core.ZERO_SLICE_STATE,
          cores: { [CORE.key]: CORE },
          order: [CORE.key],
          selected: undefined,
        },
      },
    });
    render(
      <Auth.Guard>
        <span>workspace</span>
      </Auth.Guard>,
      { wrapper },
    );
    fireEvent.click(await screen.findByText(CORE.name));
    await screen.findByText("Log in with idp-9");
    const last = buttonOf("Log in with idp-9");
    const list = scrollParent(last);
    const card = scrollParent(list);
    // the card grows to fit the form and the window, over a short transition
    await vi.waitFor(() => expect(card.scrollHeight).toBe(card.clientHeight));
    const bottom = (el: HTMLElement): number => el.getBoundingClientRect().bottom;
    list.scrollTop = 0;
    expect(bottom(buttonOf("Log in with idp-2"))).toBeCloseTo(bottom(list), 0);
    list.scrollTop = list.scrollHeight;
    expect(bottom(last)).toBeCloseTo(bottom(list), 0);
  });
});
