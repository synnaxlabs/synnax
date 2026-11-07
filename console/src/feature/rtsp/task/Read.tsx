// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack, type rtsp, type Synnax as Client } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { Rack } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { type FC } from "react";

import {
  deployReadConfigZ,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadSchemas,
} from "@/feature/rtsp/task/types";
import { CSS } from "@/platform/css";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const path = deep.path<Task.Values<ReadSchemas>>();

const URL_INPUT_PROPS = { placeholder: "rtsp://user:password@192.168.1.10:554/stream" };

const renderProtocol = Component.renderProp(
  (p: Omit<Select.SingleSimpleProps<rtsp.Protocol>, "children" | "resourceName">) => (
    <Select.Simple<rtsp.Protocol> {...p} resourceName="transport">
      <Select.Item itemKey="automatic">Automatic</Select.Item>
      <Select.Item itemKey="tcp">TCP</Select.Item>
      <Select.Item itemKey="udp">UDP</Select.Item>
    </Select.Simple>
  ),
);

const renderRack = Component.renderProp(
  (p: Omit<Rack.SelectSingleProps, "variant">) => (
    <Rack.SelectSingle {...p} query={RACK_QUERY} />
  ),
);

const RACK_QUERY: rack.RetrieveMultipleParams = {
  where: { integrations: { any: ["rtsp"] } },
};

const Properties: FC = () => (
  <Flex.Box x grow>
    <PForm.TextField
      path={path.config.url}
      label="Camera URL"
      inputProps={URL_INPUT_PROPS}
      grow
    />
    <PForm.Field path={path.config.protocol} label="Transport">
      {renderProtocol}
    </PForm.Field>
    <PForm.Field path={path.rack} label="Connect from">
      {renderRack}
    </PForm.Field>
  </Flex.Box>
);

const Form: FC = () => (
  <PForm.Sections className={CSS.B("task-sections")}>
    <PForm.Section title="Channel">
      <Task.VideoChannelField />
    </PForm.Section>
    <PForm.Section title="Timing">
      <Task.VideoTimeFields />
    </PForm.Section>
  </PForm.Sections>
);

const getInitialValues: Task.GetInitialValues<ReadSchemas> = ({ config }) => {
  const parsed = READ_SCHEMAS.config.safeParse(config ?? {});
  return {
    name: "RTSP read task",
    type: READ_TYPE,
    config: parsed.success ? parsed.data : READ_SCHEMAS.config.parse({}),
  };
};

const onConfigure: Task.OnConfigure<ReadSchemas["config"]> = async (
  client: Client,
  config,
  name,
) => [
  {
    ...config,
    channel: await Task.configureVideoChannel(
      client,
      config.channel,
      name,
      !config.dataSavingDisabled,
    ),
  },
  0,
];

export const Read = Task.wrapForm({
  Properties,
  Form,
  schemas: READ_SCHEMAS,
  deployConfigZ: deployReadConfigZ,
  type: READ_TYPE,
  onConfigure,
});

export const useCreateRead = Task.createUseCreate({ getInitialValues });

export const ReadSelectable: Selector.Selectable = {
  type: READ_TYPE,
  title: "RTSP read task",
  label: "Read",
  icon: <Icon.Video />,
  useOnSelect: useCreateRead,
};
