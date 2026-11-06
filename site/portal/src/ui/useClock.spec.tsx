// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// @vitest-environment jsdom

import { act } from "@testing-library/react";
import { type ReactElement } from "react";
import { hydrateRoot } from "react-dom/client";
import { renderToString } from "react-dom/server";
import { afterAll, describe, expect, it, vi } from "vitest";

import { useClock } from "@/ui/useClock";

// The browser clock reads the zone when its module loads, so the zone is set first.
vi.hoisted(() => vi.stubEnv("TZ", "Asia/Tokyo"));

const AT = new Date("2026-11-06T03:30:00Z");

const Time = (): ReactElement => <span>{useClock().dateTime(AT)}</span>;

describe("useClock", () => {
  afterAll(() => {
    vi.unstubAllEnvs();
  });

  it("should read UTC in the server render", () => {
    expect(renderToString(<Time />)).toBe("<span>2026-11-06 03:30 UTC</span>");
  });

  it("should read the browser's zone after hydration without a mismatch", async () => {
    const container = document.createElement("div");
    container.innerHTML = renderToString(<Time />);
    document.body.appendChild(container);
    const onRecoverableError = vi.fn();
    const root = await act(async () =>
      hydrateRoot(container, <Time />, { onRecoverableError }),
    );
    expect(onRecoverableError).not.toHaveBeenCalled();
    expect(container.textContent).toBe("2026-11-06 12:30 GMT+9");
    act(() => root.unmount());
    container.remove();
  });
});
