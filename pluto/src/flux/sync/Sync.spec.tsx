// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { newTestClient } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { act, useEffect, useState } from "react";
import { describe, expect, it } from "vitest";

import { Sync } from "@/flux/sync";
import { newSynnaxWrapper } from "@/testutil/Synnax";

describe("sync", () => {
  it("should add a basic listener", async () => {
    const client = newTestClient();
    const testChannelName = `sync_test_channel_${id.create()}`;
    await client.channels.create({
      name: testChannelName,
      dataType: "string",
      virtual: true,
    });
    const writer = await client.openWriter([testChannelName]);
    const { result, unmount } = renderHook(
      () => {
        const [data, setData] = useState<string[]>([]);
        const [open, setOpen] = useState(false);
        const addListener = Sync.useAddListener();
        useEffect(
          () =>
            addListener({
              channel: testChannelName,
              handler: (frame) => {
                setData((state) => [
                  ...state,
                  ...frame.series.flatMap((s) => s.toStrings()).flat(),
                ]);
              },
              onOpen: () => setOpen(true),
            }),
          [addListener],
        );
        return { data, open };
      },
      { wrapper: newSynnaxWrapper(client) },
    );
    await waitFor(async () => expect(result.current.open).toEqual(true));
    await act(
      async () => await writer.write({ [testChannelName]: "write number one" }),
    );
    await waitFor(async () =>
      expect(result.current.data).toEqual(["write number one"]),
    );
    await writer.close();
    unmount();
  });
});
