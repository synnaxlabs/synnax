// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Core } from "@/platform/core";
import { renderCoreUI } from "@/platform/core/testutil";
import { createCore, createCoreState } from "@/session/core/testutil";

const CORE = createCore("Alpha", { clusterKey: "cluster-9" });

describe("CopyLinkToolbarButton", () => {
  it("should render nothing when links are disabled", async () => {
    await renderCoreUI(
      <Core.CopyLinkToolbarButton
        name="My Range"
        ontologyID={{ type: "range", key: "range-key" }}
      />,
      createCoreState([CORE], CORE.key),
    );
    expect(screen.queryByRole("button")).toBeNull();
  });
});
