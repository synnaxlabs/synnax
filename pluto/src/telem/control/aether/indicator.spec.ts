// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp } from "@synnaxlabs/x";
import { afterEach, describe, expect, it } from "vitest";

import { aetherTest } from "@/aether/testutil";
import { Indicator, indicatorStateZ } from "@/telem/control/aether/indicator";
import { buildStack } from "@/testutil/providers";

describe("Indicator", () => {
  let teardown: (() => void) | null = null;
  afterEach(() => {
    teardown?.();
    teardown = null;
  });

  it("should leave the color absent when it has no color source", () => {
    const stack = buildStack({ registry: { [Indicator.TYPE]: Indicator } });
    teardown = () => stack.driver.delete([aetherTest.ROOT_KEY]);
    const path = [...stack.basePath, "indicator"];
    stack.driver.update(
      path,
      Indicator.TYPE,
      indicatorStateZ.parse({
        status: {
          key: "no_chip",
          variant: "warning",
          message: "No chip connected",
          time: TimeStamp.now(),
        },
      }),
    );
    expect(stack.driver.find<Indicator>(path).state.color).toBeUndefined();
  });
});
