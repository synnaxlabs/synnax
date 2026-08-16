// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { act, fireEvent, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, type Mock, vi } from "vitest";

import { Cluster } from "@/platform/cluster";
import { renderClusterUI } from "@/platform/cluster/testutil";
import { createClusterState } from "@/session/cluster/testutil";
import { stubClipboardWriteText } from "@/testutil";

describe("CopyLinkToolbarButton", () => {
  let writeText: Mock;
  const id: ontology.ID = { type: "range", key: "range-key" };

  beforeEach(() => {
    writeText = stubClipboardWriteText();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("should copy the resource link when clicked", async () => {
    await renderClusterUI(
      <Cluster.CopyLinkToolbarButton name="My Range" ontologyID={id} />,
      createClusterState([], "cluster-9"),
    );
    const button = screen.getByRole("button");
    await act(async () => {
      fireEvent.click(button);
    });
    expect(writeText).toHaveBeenCalledTimes(1);
    const url = writeText.mock.calls[0][0];
    expect(url).toContain("cluster-9");
    expect(url).toContain("range");
    expect(url).toContain("range-key");
  });
});
