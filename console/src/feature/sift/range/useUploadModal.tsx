// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/sift/range/UploadModal.css";

import {
  channel,
  DisconnectedError,
  type ranger,
  type status,
  TimeSpan,
} from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { type Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Channel, Synnax } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { useState } from "react";
import { z } from "zod";

import { Select as SelectDevice } from "@/feature/sift/device/Select";
import {
  SCAN_SCHEMAS,
  SCAN_TYPE,
  UPLOAD_COMMAND_TYPE,
} from "@/feature/sift/task/types";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Task } from "@/platform/task";
import { Triggers } from "@/platform/triggers";

const UPLOAD_TIMEOUT = TimeSpan.seconds(15);

// The device select sets rack, which the upload does not read.
const formSchema = z.object({
  rack: z.number(),
  config: z.object({
    device: Task.deviceKeyZ,
    asset: z.string().min(1, "Asset is required"),
    channels: channel.keyZ.array().min(1, "Select at least one channel"),
  }),
});

const path = deep.path<z.infer<typeof formSchema>>();

export interface UploadModalParams {
  rangeKey: ranger.Key;
  rangeName: string;
}

// Sift stores only persisted data channels. The Core sends each index on its own.
const CHANNEL_QUERY: Channel.ListQuery = {
  where: { virtual: false, isIndex: { eq: false }, internal: { eq: false } },
};

// The query keeps calculated channels, which the Core does not store.
const isStored = (ch: channel.Channel): boolean => !ch.isCalculated;

const CHANNEL_TRIGGER_PROPS: Select.MultipleTriggerProps<channel.Key> = {
  placeholder: "Select channels to upload",
};

const ASSET_INPUT_PROPS = { placeholder: "Test stand 1" } as const;

interface UploadButtonProps extends UploadModalParams {
  onUploaded: () => void;
  onRejection: (rejection: status.Status | null) => void;
}

const UploadButton = ({
  rangeKey,
  rangeName,
  onUploaded,
  onRejection,
}: UploadButtonProps) => {
  const client = Synnax.use();
  const handleError = Status.useAsyncErrorHandler();
  const addStatus = Status.useAdder();
  const { get, validate } = Form.useContext();
  const [pending, setPending] = useState(false);
  const handleClick = async (): Promise<void> => {
    if (!validate()) return;
    onRejection(null);
    setPending(true);
    await handleError(
      async () => {
        if (client == null) throw new DisconnectedError();
        const { device, asset, channels } = get(path.config).value;
        const dev = await client.devices.retrieve({ key: device });
        const scanTask = await client.tasks.retrieve({
          type: SCAN_TYPE,
          rack: dev.rack,
          one: true,
          schemas: SCAN_SCHEMAS,
        });
        const res = await scanTask.executeCommandSync({
          type: UPLOAD_COMMAND_TYPE,
          timeout: UPLOAD_TIMEOUT,
          args: { device, range: rangeKey, channels, asset },
        });
        // The reply of the scan task notifies a rejection on its own.
        if (res.variant === "error") return onRejection(res);
        addStatus({
          variant: "success",
          name: rangeName,
          message: "Queued upload to Sift",
        });
        onUploaded();
      },
      { message: "Failed to upload to Sift", name: rangeName },
    );
    setPending(false);
  };
  return (
    <Button.Button
      variant="filled"
      onClick={() => void handleClick()}
      status={pending ? "loading" : undefined}
      trigger={Triggers.SAVE}
      triggerIndicator
    >
      Upload
    </Button.Button>
  );
};

const Upload = ({
  rangeKey,
  rangeName,
  close,
}: Modals.ContentProps<UploadModalParams, void>) => {
  const form = Form.use<typeof formSchema>({
    schema: formSchema,
    values: { rack: 0, config: { device: "", asset: "", channels: [] } },
  });
  const [rejection, setRejection] = useState<status.Status | null>(null);
  return (
    <Form.Form<typeof formSchema> {...form}>
      <Modal.Frame className={CSS.B("sift-upload")}>
        <Modal.Header icon={<Icon.Logo.Sift />}>{HEADER}</Modal.Header>
        <Modal.Body gap="large">
          <Text.Text level="h3" weight={450}>
            Upload {rangeName} to Sift
          </Text.Text>
          <Flex.Box x gap="large">
            <SelectDevice />
            <Form.TextField
              path={path.config.asset}
              label="Asset"
              inputProps={ASSET_INPUT_PROPS}
              grow
            />
          </Flex.Box>
          <Form.Field path={path.config.channels} label="Channels">
            {({ value, onChange }) => (
              <Channel.SelectMultiple
                value={value}
                onChange={onChange}
                query={CHANNEL_QUERY}
                filter={isStored}
                triggerProps={CHANNEL_TRIGGER_PROPS}
                full="x"
              />
            )}
          </Form.Field>
        </Modal.Body>
        <Modal.Footer>
          {rejection != null && (
            <Nav.Bar.Start>
              <Status.Summary
                variant="error"
                message={rejection.description ?? rejection.message}
              />
            </Nav.Bar.Start>
          )}
          <Nav.Bar.End x align="center">
            <UploadButton
              rangeKey={rangeKey}
              rangeName={rangeName}
              onUploaded={close}
              onRejection={setRejection}
            />
          </Nav.Bar.End>
        </Modal.Footer>
      </Modal.Frame>
    </Form.Form>
  );
};

const HEADER = ["Range", "Upload to Sift"];

export const useUploadModal = Modals.create<UploadModalParams>(Upload);
