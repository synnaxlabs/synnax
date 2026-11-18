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
  type device,
  DisconnectedError,
  type ranger,
  type status,
  TimeRange,
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
import { Telem } from "@synnaxlabs/lyra/telem";
import { Text } from "@synnaxlabs/lyra/text";
import { Channel, Device, Synnax } from "@synnaxlabs/pluto";
import { type CrudeTimeRange, deep } from "@synnaxlabs/x";
import { useState } from "react";
import { z } from "zod";

import { Asset } from "@/feature/sift/asset";
import { MAKE } from "@/feature/sift/device/types";
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
  /** The name the modal and its statuses show. */
  name: string;
  /** The key of the range to upload, or a plain time range. */
  source: ranger.Key | CrudeTimeRange;
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

// Case conversion drops the JSON form of a TimeRange, so the span goes as bare values.
const encodeTimeRange = (tr: CrudeTimeRange) => {
  const { start, end } = new TimeRange(tr);
  return { start: start.valueOf(), end: end.valueOf() };
};

interface UploadButtonProps extends UploadModalParams {
  onUploaded: () => void;
  onRejection: (rejection: status.Status | null) => void;
}

const UploadButton = ({ name, source, onUploaded, onRejection }: UploadButtonProps) => {
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
        const data =
          typeof source === "string"
            ? { range: source }
            : { timeRange: encodeTimeRange(source) };
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
          args: { device, channels, asset, ...data },
        });
        // The reply of the scan task notifies a rejection on its own.
        if (res.variant === "error") return onRejection(res);
        addStatus({
          variant: "success",
          name,
          message: "Queued upload to Sift",
        });
        onUploaded();
      },
      { message: "Failed to upload to Sift", name },
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

interface UploadFormProps extends Modals.ContentProps<UploadModalParams, void> {
  device: device.Key;
}

const UploadForm = ({ name, source, close, device }: UploadFormProps) => {
  const form = Form.use<typeof formSchema>({
    schema: formSchema,
    values: { rack: 0, config: { device, asset: "", channels: [] } },
  });
  const [rejection, setRejection] = useState<status.Status | null>(null);
  return (
    <Form.Form<typeof formSchema> {...form}>
      <Modal.Frame className={CSS.B("sift-upload")}>
        <Modal.Header icon={<Icon.Logo.Sift />}>
          {typeof source === "string" ? RANGE_HEADER : SPAN_HEADER}
        </Modal.Header>
        <Modal.Body gap="large">
          <Flex.Box y gap="tiny">
            <Text.Text level="h3" weight={450}>
              Upload {name} to Sift
            </Text.Text>
            {typeof source !== "string" && (
              <Telem.Text.TimeRange>{source}</Telem.Text.TimeRange>
            )}
          </Flex.Box>
          <Flex.Box x gap="large">
            <Asset.Destination
              path={path.config.asset}
              devicePath={path.config.device}
              grow
            />
          </Flex.Box>
          <Form.Field path={path.config.channels} label="Channels" required>
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
              name={name}
              source={source}
              onUploaded={close}
              onRejection={setRejection}
            />
          </Nav.Bar.End>
        </Modal.Footer>
      </Modal.Frame>
    </Form.Form>
  );
};

// Two at most: the modal picks the connection only when it is the only one.
const DEVICES_QUERY = { where: { make: { eq: MAKE } }, limit: 2 };

const Upload = (props: Modals.ContentProps<UploadModalParams, void>) => {
  const { variant, data } = Device.useResultMultiple(DEVICES_QUERY);
  if (variant === "loading") return null;
  const device = data?.length === 1 ? data[0].key : "";
  return <UploadForm {...props} device={device} />;
};

const RANGE_HEADER = ["Range", "Upload to Sift"];
const SPAN_HEADER = ["Time range", "Upload to Sift"];

export const useUploadModal = Modals.create<UploadModalParams>(Upload);
