// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { renderToString } from "react-dom/server";
import { afterEach, describe, expect, it, vi } from "vitest";

import { DeferredScene } from "@/components/DeferredScene";

const Scene = ({ stage }: { stage: number }) => <svg aria-label={`Stage ${stage}`} />;

describe("DeferredScene", () => {
  afterEach(cleanup);

  it("keeps geometry and its module request out of the server render", () => {
    const load = vi.fn().mockResolvedValue({ default: Scene });
    const html = renderToString(
      <DeferredScene load={load} sceneProps={{ stage: 0 }} viewBox="0 0 900 400" />,
    );
    expect(load).not.toHaveBeenCalled();
    expect(html).toContain("aspect-ratio:900 / 400");
    expect(html).not.toContain("<svg");
  });

  it("renders the latest state if a visitor changes steps before loading completes", async () => {
    const loaded = Promise.withResolvers<{ default: typeof Scene }>();
    const load = vi.fn(() => loaded.promise);
    const { rerender } = render(
      <DeferredScene load={load} sceneProps={{ stage: 0 }} viewBox="0 0 900 400" />,
    );
    rerender(
      <DeferredScene load={load} sceneProps={{ stage: 2 }} viewBox="0 0 900 400" />,
    );
    loaded.resolve({ default: Scene });
    expect(await screen.findByLabelText("Stage 2")).toBeTruthy();
    expect(load).toHaveBeenCalledTimes(1);
  });

  it("lets a failed chunk request be retried without losing the selected step", async () => {
    const load = vi
      .fn()
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce({ default: Scene });
    render(
      <DeferredScene load={load} sceneProps={{ stage: 1 }} viewBox="0 0 900 400" />,
    );
    fireEvent.click(await screen.findByRole("button", { name: /Reload diagram/ }));
    expect(await screen.findByLabelText("Stage 1")).toBeTruthy();
    expect(load).toHaveBeenCalledTimes(2);
  });
});
