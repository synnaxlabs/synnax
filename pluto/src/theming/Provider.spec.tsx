// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Profiler } from "react";
import { describe, expect, it } from "vitest";

import { render } from "@/testutil";
import { Theming } from "@/theming";

describe("Theming.Provider", () => {
  it("should not re-render when it hands the theme to the worker", () => {
    let renders = 0;
    render(
      <Profiler id="theming" onRender={() => renders++}>
        <Theming.Provider>
          <div />
        </Theming.Provider>
      </Profiler>,
    );
    expect(renders).toBe(1);
  });
});
