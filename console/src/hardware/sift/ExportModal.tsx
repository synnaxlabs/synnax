// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/sift/ExportModal.css";

import { channel } from "@synnaxlabs/client";
import {
  Button,
  Channel,
  Flex,
  Form,
  Icon,
  Input,
  Nav,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { type CrudeTimeRange, TimeRange } from "@synnaxlabs/x";
import { useState } from "react";
import { z } from "zod";

import { Device } from "@/hardware/sift/device";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

export interface UploadModalArgs extends Modals.BaseArgs<void> {
  timeRange: CrudeTimeRange;
  channels: channel.Keys;
  name: string;
}

export const UPLOAD_MODAL_LAYOUT_TYPE = "exportToSift";

const formSchema = z.object({
  name: z.string(),
  deviceKey: z.string().min(1, "Please select a Sift device"),
  assetName: z.string().min(1, "Asset name is required"),
  runName: z.string().min(1, "Run name is required"),
  channels: channel.keyZ.array().min(1, "Please select at least one channel"),
  timeRange: z.object({
    start: z.number(),
    end: z.number(),
  }),
});

interface UploadButtonProps {
  handleFinish: () => void;
  timeRange: CrudeTimeRange;
}

const UploadButton = ({ handleFinish, timeRange }: UploadButtonProps) => {
  const client = Synnax.use();
  const addStatus = Status.useAdder();
  const { get } = Form.useContext();
  const [loading, setLoading] = useState(false);

  const handleClick = async () => {
    if (client == null) return;

    const deviceKey = get<string>("deviceKey").value;
    const assetName = get<string>("assetName").value;
    const runName = get<string>("runName").value;
    const channels = get<channel.Keys>("channels").value;

    if (!deviceKey || !assetName || !runName || channels.length === 0) {
      addStatus({
        variant: "error",
        message: "Please fill in all required fields",
      });
      return;
    }

    setLoading(true);

    try {
      const tr = new TimeRange(timeRange);
      const taskConfig = {
        device_key: deviceKey,
        asset_name: assetName,
        run_name: runName,
        channels,
        time_range: {
          start: tr.start.valueOf(),
          end: tr.end.valueOf(),
        },
      };

      const embeddedRack = await client.racks.retrieve({ name: "Node 1" });
      await embeddedRack.createTask({
        name: `Upload ${runName} to Sift`,
        type: "sift_upload",
        config: taskConfig,
      });

      addStatus({
        variant: "success",
        message: `Upload to Sift started for ${runName}`,
      });

      handleFinish();
    } catch (e) {
      addStatus({
        variant: "error",
        message: `Failed to start upload: ${String(e)}`,
      });
    } finally {
      setLoading(false);
    }
  };

  const deviceKey = Form.useFieldValue<string>("deviceKey");
  const channels = Form.useFieldValue<channel.Keys>("channels");
  const isDisabled = !deviceKey || channels.length === 0;

  return (
    <Button.Button
      variant="filled"
      disabled={isDisabled}
      status={loading ? "loading" : undefined}
      onClick={() => void handleClick()}
      trigger={Triggers.SAVE}
    >
      <Icon.Export />
      Upload to Sift
    </Button.Button>
  );
};

export const [useUploadModal, UploadModal] = Modals.createBase<void, UploadModalArgs>(
  "Upload.Sift",
  UPLOAD_MODAL_LAYOUT_TYPE,
  ({ value: { timeRange, channels, name }, onFinish }) => {
    const tr = new TimeRange(timeRange);
    const form = Form.use<typeof formSchema>({
      schema: formSchema,
      values: {
        name,
        deviceKey: "",
        assetName: "",
        runName: name,
        channels,
        timeRange: tr.numeric,
      },
    });

    const footer = (
      <>
        <Triggers.SaveHelpText action="Upload" />
        <Nav.Bar.End x align="center">
          <UploadButton handleFinish={onFinish} timeRange={timeRange} />
        </Nav.Bar.End>
      </>
    );

    return (
      <Form.Form<typeof formSchema> {...form}>
        <Modals.ModalContentLayout footer={footer} gap="huge">
          <Flex.Box y gap="small">
            <Text.Text level="h3" weight={450}>
              Upload {name} to Sift
            </Text.Text>
          </Flex.Box>
          <Flex.Box y full="x" gap="medium">
            <Flex.Box x gap="medium">
              <Device.Select />
            </Flex.Box>
            <Flex.Box x gap="medium">
              <Form.Field<string> path="assetName" label="Asset Name" required grow>
                {(p) => <Input.Text placeholder="Asset" {...p} />}
              </Form.Field>
            </Flex.Box>
            <Form.Field<string> path="runName" label="Run Name" required>
              {(p) => <Input.Text placeholder="Run" {...p} />}
            </Form.Field>
            <Form.Field<channel.Keys> path="channels" label="Channels" required>
              {({ value, onChange }) => (
                <Channel.SelectMultiple
                  value={value}
                  onChange={onChange}
                  full="x"
                  triggerProps={{ placeholder: "Select channels to upload" }}
                />
              )}
            </Form.Field>
          </Flex.Box>
        </Modals.ModalContentLayout>
      </Form.Form>
    );
  },
  { window: { resizable: false, size: { height: 520, width: 700 }, navTop: true } },
);
