// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type Synnax as Client } from "@synnaxlabs/client";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type Select } from "@synnaxlabs/lyra/select";
import { Channel } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { type FC } from "react";

import { Asset } from "@/feature/sift/asset";
import { Select as SelectDevice } from "@/feature/sift/device/Select";
import { MAKE } from "@/feature/sift/device/types";
import {
  deployStreamConfigZ,
  STREAM_SCHEMAS,
  STREAM_TYPE,
  type StreamSchemas,
} from "@/feature/sift/task/types";
import { Label } from "@/platform/label";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const path = deep.path<Task.Values<StreamSchemas>>();

const Properties = () => (
  <>
    <SelectDevice />
    <Asset.Field path={path.config.asset} devicePath={path.config.device} grow />
  </>
);

// Sift stores only persisted data channels. The Core sends each index on its own.
const CHANNEL_QUERY: Channel.ListQuery = {
  where: { virtual: false, isIndex: { eq: false }, internal: { eq: false } },
};

const CHANNEL_TRIGGER_PROPS: Select.MultipleTriggerProps<channel.Key> = {
  placeholder: "Select channels to send",
};

const Form: FC = () => (
  <PForm.Sections>
    <PForm.Section title="Data">
      <PForm.Field path={path.config.channels} label="Channels" required>
        {({ value, onChange }) => (
          <Channel.SelectMultiple
            value={value}
            onChange={onChange}
            query={CHANNEL_QUERY}
            triggerProps={CHANNEL_TRIGGER_PROPS}
            full="x"
          />
        )}
      </PForm.Field>
    </PForm.Section>
    <PForm.Section title="Runs">
      <PForm.Field
        path={path.config.labels}
        label="Labels"
        helpText="A range with one of these labels becomes a Sift run."
      >
        {({ value, onChange }) => (
          <Label.SelectMultiple value={value} onChange={onChange} full="x" />
        )}
      </PForm.Field>
    </PForm.Section>
  </PForm.Sections>
);

const getInitialValues: Task.GetInitialValues<StreamSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = STREAM_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  return { name: "Sift stream task", type: STREAM_TYPE, config: cfg };
};

const onConfigure: Task.OnConfigure<StreamSchemas["config"]> = async (
  client: Client,
  config,
) => {
  const dev = await client.devices.retrieve({ key: config.device });
  return [config, dev.rack];
};

export const Stream = Task.wrapForm({
  Properties,
  Form,
  schemas: STREAM_SCHEMAS,
  deployConfigZ: deployStreamConfigZ,
  type: STREAM_TYPE,
  onConfigure,
});

export const useCreateStream = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const StreamSelectable: Selector.Selectable = {
  type: STREAM_TYPE,
  title: "Sift stream task",
  label: "Stream",
  icon: <Icon.Logo.Sift />,
  useOnSelect: useCreateStream,
};
