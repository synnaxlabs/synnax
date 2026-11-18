// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType, schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { deep, id } from "@synnaxlabs/x";
import { fireEvent, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { StringDisplay } from "@/schematic/node/general/stringDisplay";
import { createAsyncSynnaxWrapper, renderForm } from "@/testutil";

const path = deep.path<schematic.StringDisplayNodeConfig>();

const client = createTestClient();

describe("StringDisplay form", () => {
  let wrapper: FC<PropsWithChildren>;
  let ch: channel.Channel;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    ch = await client.channels.create({
      name: id.create(),
      dataType: DataType.STRING,
      virtual: true,
    });
  });

  it("should store 0 when the channel is cleared", async () => {
    const { findByLabelText, value } = renderForm(
      <StringDisplay.spec.Form tab="telemetry" onTabChange={vi.fn()} />,
      schematic.stringDisplayNodeConfigZ.parse({
        variant: "string_display",
        channel: ch.key,
      }),
      schematic.stringDisplayNodeConfigZ,
      wrapper,
    );
    fireEvent.click(await findByLabelText(`Clear ${ch.name}`));
    await waitFor(() => expect(value(path.channel)).toBe(0));
  });
});
