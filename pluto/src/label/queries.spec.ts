// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { newTestClient } from "@synnaxlabs/client";
import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Label } from "@/label";
// import { SyncProvider } from "@/testutil/Sync";

// describe("Label.use", () => {
//   it("should return the correct labels for an id and parse new relationships", async () => {
//     const client = newTestClient();
//     const rng = await client.ranges.create({
//       name: "test",
//       timeRange: { start: 1, end: 1000 },
//     });
//     const id = rng.ontologyID;
//     const label1 = await client.labels.create({ name: "test label", color: "#FFFFFF" });
//     await client.labels.label(id, [label1.key]);
//     const { result } = renderHook(() => Label.useLabelsOf(id), {
//       wrapper: SyncProvider,
//     });
//     await waitFor(() => expect(result.current).toEqual([label1]));

//     // it should parse new relationships
//     let label2 = await client.labels.create({
//       name: "test label 2",
//       color: "#FFFFFF",
//     });
//     await client.labels.label(id, [label2.key]);
//     await waitFor(() => expect(result.current).toEqual([label1, label2]));

//     // it should parse deleted relationships
//     await client.labels.removeLabels(id, [label1.key]);
//     await waitFor(() => expect(result.current).toEqual([label2]));

//     // it should allow for labels to be updated
//     label2 = await client.labels.create({ ...label2, color: "#000000" });
//     await waitFor(() => expect(result.current).toEqual([label2]));

//     // it should allow for labels to be deleted
//     await client.labels.delete([label2.key]);
//     await waitFor(() => expect(result.current).toEqual([]));
//   });
// });
