// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { scheduler } from "@synnaxlabs/x";
import { act, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { mockCLI, renderStar } from "@/platform/github/testutil";

describe("GitHub.StarButton", () => {
  it("should not ask for a star outside Synnax Desktop", async () => {
    const commands = mockCLI("not_starred");
    await renderStar();
    await act(async () => await scheduler.flushTaskQueue());
    expect(commands).not.toHaveBeenCalledWith("github_star_status", expect.anything());
    expect(screen.queryByRole("button", { name: /star/i })).toBeNull();
  });
});
