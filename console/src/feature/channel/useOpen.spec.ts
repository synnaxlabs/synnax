// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { useOpen } from "@/feature/channel/useOpen";
import { createTestSink, wrapWithSink } from "@/platform/analytics/testutil";
import { createResource } from "@/platform/tree/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, uniqueName } from "@/testutil";

const client = createTestClient();

describe("useOpen", () => {
  it("should report the line plot it created for a channel", async () => {
    const ch = await client.channels.create({
      name: uniqueName("ch"),
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    const proj = await client.projects.create({ name: uniqueName("proj") });
    const { wrapper, store } = await createConsoleWrapper({ client });
    store.dispatch(Session.Project.select(proj.key));
    const analytics = createTestSink();
    const { result } = renderHook(useOpen, {
      wrapper: wrapWithSink(wrapper, analytics),
    });
    act(() => {
      result.current(createResource(channel.ontologyID(ch.key), ch.name, {}));
    });
    await waitFor(() =>
      expect(analytics.capture).toHaveBeenCalledWith("plot_created", {}),
    );
  });
});
