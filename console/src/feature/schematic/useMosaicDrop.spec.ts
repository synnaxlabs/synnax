// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Schematic } from "@/feature/schematic";
import { client, createSchematic } from "@/feature/schematic/testutil";
import { Session } from "@/session";
import { renderHookWithConsole } from "@/testutil";

describe("Schematic.useMosaicDrop", () => {
  it("places the schematic into the target mosaic node", async () => {
    const s = await createSchematic();
    const { result, store } = await renderHookWithConsole(
      () => Schematic.useMosaicDrop(),
      { client },
    );
    result.current({
      id: schematic.ontologyID(s.key),
      nodeKey: 3,
      location: "center",
    });
    await waitFor(() =>
      expect(Session.Layout.select(store.getState(), s.key)?.name).toBe(s.name),
    );
    expect(Session.Layout.select(store.getState(), s.key)?.tab).toMatchObject({
      mosaicKey: 3,
      location: "center",
    });
  });
});
