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
import { id } from "@synnaxlabs/x";
import { fireEvent, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { Node } from "@/schematic/node";
import { ButtonForm } from "@/schematic/node/general/button/Form";
import { createAsyncSynnaxWrapper, renderForm } from "@/testutil";

const client = createTestClient();

describe("ButtonForm", () => {
  let wrapper: FC<PropsWithChildren>;
  let ch: channel.Channel;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    ch = await client.channels.create({
      name: id.create(),
      dataType: DataType.UINT8,
      virtual: true,
    });
  });

  it("should unbind the command channel when it is cleared", async () => {
    const config = Node.createConfig({ variant: "button", commandChannel: ch.key });
    const { findByLabelText, value } = renderForm(
      <ButtonForm tab="control" onTabChange={vi.fn()} />,
      config,
      schematic.buttonNodeConfigZ,
      wrapper,
    );
    fireEvent.click(await findByLabelText(`Clear ${ch.name}`));
    await waitFor(() => expect(value("commandChannel")).toBeUndefined());
  });
});
