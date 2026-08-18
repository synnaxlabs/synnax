// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Menu as PMenu } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Range } from "@/platform/range";
import { type Session } from "@/session";
import { renderWithConsole } from "@/testutil";

const range = (persisted: boolean): Session.Range.State => ({
  key: "r1",
  name: "Alpha",
  persisted,
  variant: "static",
  timeRange: { start: 0, end: 1 },
});

describe("Range.SnapshotMenuItem", () => {
  it("should render a snapshot item for a persisted range and fire onClick", async () => {
    const onClick = vi.fn();
    await renderWithConsole(
      <PMenu.Menu>
        <Range.SnapshotMenuItem range={range(true)} onClick={onClick} />
      </PMenu.Menu>,
    );
    const item = await screen.findByText("Snapshot to Alpha");
    fireEvent.click(item);
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("should render nothing for a non-persisted range", async () => {
    await renderWithConsole(
      <PMenu.Menu>
        <Range.SnapshotMenuItem range={range(false)} onClick={vi.fn()} />
      </PMenu.Menu>,
    );
    await waitFor(() => expect(screen.queryByText(/Snapshot to/)).toBeNull());
  });

  it("should render nothing when no range is provided", async () => {
    await renderWithConsole(
      <PMenu.Menu>
        <Range.SnapshotMenuItem range={null} onClick={vi.fn()} />
      </PMenu.Menu>,
    );
    await waitFor(() => expect(screen.queryByText(/Snapshot to/)).toBeNull());
  });
});
