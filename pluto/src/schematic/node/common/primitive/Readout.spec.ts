// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component } from "@synnaxlabs/lyra/component";
import { text } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Primitive } from "@/schematic/node/common/primitive";
import { SIZE_LEVELS } from "@/schematic/node/common/size";

describe("READOUT_LEVELS", () => {
  it.each(Component.SIZES)("should set a %s readout below its field's text", (size) =>
    expect(text.LEVELS.indexOf(Primitive.READOUT_LEVELS[size])).toBeGreaterThan(
      text.LEVELS.indexOf(SIZE_LEVELS[size]),
    ),
  );
});
