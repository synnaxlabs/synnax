// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { fireEvent, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Source } from "@/arc/graph/node/source";
import { Form } from "@/arc/graph/node/source/Form";
import { createAsyncSynnaxWrapper, renderForm } from "@/testutil";

const client = createTestClient();

describe("source.Form", () => {
  let wrapper: FC<PropsWithChildren>;
  let ch: channel.Channel;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    ch = await client.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
  });

  it("should store no channel when the channel is cleared", async () => {
    const { findByLabelText, value } = renderForm(
      <Form />,
      { type: "on", channel: ch.key },
      Source.configZ,
      wrapper,
    );
    fireEvent.click(await findByLabelText(`Clear ${ch.name}`));
    await waitFor(() => expect(value("channel")).toBe(0));
  });
});
