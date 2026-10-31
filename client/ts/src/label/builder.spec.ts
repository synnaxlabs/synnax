// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { filter } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { label } from "@/label";
import { type ontology } from "@/ontology";

describe("builder", () => {
  it("should export a line named by a reserved word", () => {
    const id: ontology.ID = { type: "range", key: "a" };
    expect(label.retrieveRequestZ.parse(filter.toRequest([label.for(id)]))).toEqual({
      where: { for: id },
    });
  });
});
