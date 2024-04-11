// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useState, type ReactElement } from "react";

import { Align, Button, Device, Form, Input, Synnax } from "@synnaxlabs/pluto";
import { useMutation } from "@tanstack/react-query";

import { CSS } from "@/css";
import { ChannelList } from "@/hardware/opcua/ChannelList";
import { readTaskConfigZ } from "@/hardware/opcua/types";
import { type Layout } from "@/layout";

export const readTaskLayout: Layout.LayoutState = {
  name: "Configure OPC UA Read Task",
  key: "readOPCUATask",
  type: "readOPCUATask",
  windowKey: "readOPCUATask",
  location: "mosaic",
};

export const ReadTask = (): ReactElement => {
  const client = Synnax.use();
  const [taskKey, setTaskKey] = useState<string | undefined>(undefined);
  const methods = Form.use({
    schema: readTaskConfigZ,
    values: {
      device: "",
      sampleRate: 50,
      streamRate: 25,
      channels: [],
    },
  });

  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);
  const [selectedChannelIndex, setSelectedChannelIndex] = useState<number | null>(null);

  const configure = useMutation({
    mutationKey: [client?.key],
    mutationFn: async () => {
      if (client == null) return;
      const rack = await client.hardware.racks.retrieve("sy_node_1_rack");
      const t = await rack.createTask({
        key: taskKey,
        name: "OPCUA Read Task",
        type: "opcuaReader",
        config: methods.value(),
      });
      setTaskKey(t.key);
    },
  });

  return (
    <Align.Space className={CSS.B("opcua-read-task")} direction="y" grow empty>
      <Form.Form {...methods}>
        <Align.Space direction="x">
          <Form.Field<string> path="device" label="Device">
            {(p) => <Device.SelectSingle {...p} />}
          </Form.Field>
          <Form.Field<number> label="Sample Rate" path="sampleRate">
            {(p) => <Input.Numeric {...p} />}
          </Form.Field>
          <Form.Field<number> label="Stream Rate" path="streamRate">
            {(p) => <Input.Numeric {...p} />}
          </Form.Field>
        </Align.Space>
        <Align.Space direction="x">
          <ChannelList
            path="channels"
            selected={selectedChannels}
            onSelect={useCallback(
              (v, i) => {
                setSelectedChannels(v);
                setSelectedChannelIndex(i);
              },
              [setSelectedChannels, setSelectedChannelIndex],
            )}
          />
        </Align.Space>
      </Form.Form>
      <Button.Button onClick={() => configure.mutate()}>Configure</Button.Button>
    </Align.Space>
  );
};
