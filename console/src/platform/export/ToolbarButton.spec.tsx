// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { type Status } from "@synnaxlabs/pluto";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Export } from "@/platform/export";
import { CaptureStatuses, renderWithConsole } from "@/testutil";

describe("Export.ToolbarButton", () => {
  it("starts the export flow for the given ID when clicked", async () => {
    let statuses: Status.NotificationSpec[] = [];
    await renderWithConsole(
      <>
        <Export.ToolbarButton id={log.ontologyID("k")} />
        <CaptureStatuses onStatuses={(next) => (statuses = next)} />
      </>,
    );
    fireEvent.click(screen.getByRole("button"));
    // No Core is connected, so reaching the flow's disconnected error proves the
    // click dispatched the export.
    await waitFor(() =>
      expect(statuses.map((s) => s.message)).toContain("Failed to export resource"),
    );
  });

  it("forwards the disabled prop, suppressing the click", async () => {
    let statuses: Status.NotificationSpec[] = [];
    await renderWithConsole(
      <>
        <Export.ToolbarButton id={log.ontologyID("k")} disabled />
        <CaptureStatuses onStatuses={(next) => (statuses = next)} />
      </>,
    );
    fireEvent.click(screen.getByRole("button"));
    await act(async () => {});
    expect(statuses).toHaveLength(0);
  });
});
