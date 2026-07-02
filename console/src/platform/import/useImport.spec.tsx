// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, waitFor } from "@testing-library/react";
import { type ReactElement, useEffect } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Import } from "@/platform/import";
import { Runtime } from "@/platform/runtime";
import { renderWithConsole } from "@/testutil/testutil";

// pickFiles opens the native (Tauri) / browser file-picker dialog, an unmockable
// runtime seam with no injection point. Stub it to hand the import flow real file
// contents; everything downstream (parse, ingester dispatch) runs for real.
const pickFiles = vi.spyOn(Runtime, "pickFiles");

afterEach(() => {
  vi.restoreAllMocks();
});

interface HarnessProps {
  fileIngesters: Import.FileIngesters;
  onReady: (run: (projectKey?: string) => void) => void;
}

const Inner = ({ onReady }: Pick<HarnessProps, "onReady">): ReactElement => {
  const run = Import.useImport();
  useEffect(() => onReady(run), [onReady, run]);
  return <span>ready</span>;
};
Inner.displayName = "Inner";

const Harness = ({ fileIngesters, onReady }: HarnessProps): ReactElement => (
  <Import.FileIngestersProvider fileIngesters={fileIngesters}>
    <Inner onReady={onReady} />
  </Import.FileIngestersProvider>
);
Harness.displayName = "Harness";

describe("useImport", () => {
  it("reads each picked file and dispatches its parsed contents to the ingester", async () => {
    pickFiles.mockResolvedValue([
      { name: "widget.json", path: "widget.json", read: async () => '{"type":"log"}' },
    ]);
    const log = vi.fn();
    let run: ((projectKey?: string) => void) | undefined;
    await renderWithConsole(
      <Harness fileIngesters={{ log }} onReady={(r) => (run = r)} />,
      { preloadedState: { project: { version: 0, selected: "project-1" } } },
    );
    await waitFor(() => expect(run).toBeDefined());
    act(() => run?.());
    await waitFor(() => expect(log).toHaveBeenCalledTimes(1));
    expect(log.mock.calls[0][0]).toEqual({ type: "log" });
  });

  it("does nothing when the file picker is cancelled", async () => {
    pickFiles.mockResolvedValue(null);
    const log = vi.fn();
    let run: ((projectKey?: string) => void) | undefined;
    await renderWithConsole(
      <Harness fileIngesters={{ log }} onReady={(r) => (run = r)} />,
      { preloadedState: { project: { version: 0, selected: "project-1" } } },
    );
    await waitFor(() => expect(run).toBeDefined());
    act(() => run?.());
    await waitFor(() => expect(pickFiles).toHaveBeenCalled());
    expect(log).not.toHaveBeenCalled();
  });
});
