// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Range } from "@/feature/range";
import { renderHookWithConsole } from "@/testutil";

const client = createTestClient();

describe("range/palette", () => {
  it("shows both commands to a user with range permissions", async () => {
    const { result } = await renderHookWithConsole(
      () => ({
        create: Range.CreateCommand.useVisible?.() ?? false,
        explorer: Range.OpenExplorerCommand.useVisible?.() ?? false,
      }),
      { client },
    );
    await waitFor(() => {
      expect(result.current.create).toBe(true);
      expect(result.current.explorer).toBe(true);
    });
  });
});
