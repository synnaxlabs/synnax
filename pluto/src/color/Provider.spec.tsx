// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, waitForStreamLive } from "@synnaxlabs/client/testutil";
import { Color as Base } from "@synnaxlabs/lyra/color";
import { renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Color } from "@/color";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("Color.Provider", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    const Synnax = await createAsyncSynnaxWrapper({ client });
    wrapper = ({ children }) => (
      <Synnax>
        <Color.Provider>{children}</Color.Provider>
      </Synnax>
    );
    await waitForStreamLive(client.connection);
  });

  it("should offer the series colors as the presets of every picker", async () => {
    const { result } = renderHook(
      () => ({ presets: Base.usePresets(), series: Color.useSeriesValues() }),
      { wrapper },
    );
    await waitFor(() => {
      expect(result.current.series.length).toBeGreaterThan(0);
      expect(result.current.presets).toEqual(result.current.series);
    });
  });
});
