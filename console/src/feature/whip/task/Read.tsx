// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack, type Synnax as Client } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";
import { Rack } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { type FC } from "react";

import {
  publishURL,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadSchemas,
} from "@/feature/whip/task/types";
import { CSS } from "@/platform/css";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";
import { Session } from "@/session";

const path = deep.path<Task.Values<ReadSchemas>>();

const renderRack = Component.renderProp(
  (p: Omit<Rack.SelectSingleProps, "variant">) => (
    <Rack.SelectSingle {...p} query={RACK_QUERY} />
  ),
);

const RACK_QUERY: rack.RetrieveMultipleParams = {
  where: { integrations: { any: ["whip"] } },
};

const Properties: FC = () => (
  <PForm.Field path={path.rack} label="Receive on">
    {renderRack}
  </PForm.Field>
);

const URL_HELP =
  "Paste this URL into the encoder, and set its bearer token to an API key that " +
  "may write to the channel. In OBS, choose the WHIP service in the stream settings.";

const PublishURL: FC = () => {
  const key = Task.useKey();
  const core = Session.Core.useSelectSelected();
  const url = key == null || core == null ? "" : publishURL(core, key);
  return (
    <Input.Item label="URL" helpText={URL_HELP}>
      <Flex.Box x align="center" gap="small">
        <Text.Text level="p" color={10} overflow="fade" grow>
          {url}
        </Text.Text>
        <Button.Copy
          text={url}
          disabled={url === ""}
          tooltip="Copy URL"
          variant="outlined"
          size="small"
        >
          <Icon.Copy />
        </Button.Copy>
      </Flex.Box>
    </Input.Item>
  );
};

const Form: FC = () => (
  <PForm.Sections className={CSS.B("task-sections")}>
    <PForm.Section title="Channel">
      <Task.VideoChannelField />
    </PForm.Section>
    <PForm.Section title="Timing">
      <Task.VideoTimeFields />
    </PForm.Section>
    <PForm.Section title="Encoder">
      <PublishURL />
    </PForm.Section>
  </PForm.Sections>
);

const getInitialValues: Task.GetInitialValues<ReadSchemas> = ({ config }) => {
  const parsed = READ_SCHEMAS.config.safeParse(config ?? {});
  return {
    name: "WHIP read task",
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
  deployConfigZ: READ_SCHEMAS.config,
  type: READ_TYPE,
  onConfigure,
});

export const useCreateRead = Task.createUseCreate({ getInitialValues });

export const ReadSelectable: Selector.Selectable = {
  type: READ_TYPE,
  title: "WHIP read task",
  label: "Read",
  icon: <Icon.Video />,
  useOnSelect: useCreateRead,
};
