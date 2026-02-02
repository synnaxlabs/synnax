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
  List,
  Nav,
  Select,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { type CrudeTimeRange, TimeRange } from "@synnaxlabs/x";
import { useEffect, useState } from "react";
import { z } from "zod";

import { MAKE, type Properties } from "@/hardware/sift/device/types";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

export interface ExportModalArgs extends Modals.BaseArgs<void> {
  timeRange: CrudeTimeRange;
  channels: channel.Keys;
  name: string;
}

export const EXPORT_MODAL_LAYOUT_TYPE = "exportToSift";

const formSchema = z.object({
  name: z.string(),
  deviceKey: z.string().min(1, "Please select a Sift device"),
  assetName: z.string().min(1, "Asset name is required"),
  flowName: z.string().min(1, "Flow name is required"),
  runName: z.string().min(1, "Run name is required"),
  channels: channel.keyZ.array().min(1, "Please select at least one channel"),
  timeRange: z.object({
    start: z.number(),
    end: z.number(),
  }),
});

interface SiftDeviceSelectProps {
  value: string;
  onChange: (value: string) => void;
}

interface SiftDevice {
  key: string;
  name: string;
}

const SiftDeviceSelect = ({ value, onChange }: SiftDeviceSelectProps) => {
  const client = Synnax.use();
  const [devices, setDevices] = useState<SiftDevice[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (client == null) {
      setLoading(false);
      return;
    }
    client.devices
      .retrieve<Properties>({ makes: [MAKE] })
      .then((devs) => setDevices(devs.map((d) => ({ key: d.key, name: d.name }))))
      .catch(() => setDevices([]))
      .finally(() => setLoading(false));
  }, [client]);

  const { data, getItem } = List.useStaticData<string, SiftDevice>({ data: devices });
  if (loading) return <Text.Text level="small">Loading devices...</Text.Text>;

  if (devices.length === 0)
    return (
      <Text.Text level="small" color={8}>
        No Sift devices found. Please connect a Sift device first.
      </Text.Text>
    );

  return (
    <Select.Single<string, SiftDevice>
      value={value}
      onChange={onChange}
      allowNone={false}
      data={data}
      getItem={getItem}
      resourceName="Sift device"
    >
      {(props: List.ItemProps<string>) => {
        const dev = getItem(props.itemKey);
        if (dev == null) return null;
        return <Select.ListItem {...props}>{dev.name}</Select.ListItem>;
      }}
    </Select.Single>
  );
};

interface ExportButtonProps {
  handleFinish: () => void;
  timeRange: CrudeTimeRange;
}

const ExportButton = ({ handleFinish, timeRange }: ExportButtonProps) => {
  const client = Synnax.use();
  const addStatus = Status.useAdder();
  const { get } = Form.useContext();
  const [loading, setLoading] = useState(false);

  const handleClick = async () => {
    if (client == null) return;

    const deviceKey = get<string>("deviceKey").value;
    const assetName = get<string>("assetName").value;
    const flowName = get<string>("flowName").value;
    const runName = get<string>("runName").value;
    const channels = get<channel.Keys>("channels").value;

    if (!deviceKey || !assetName || !flowName || !runName || channels.length === 0) {
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
        flow_name: flowName,
        run_name: runName,
        channels,
        time_range: {
          start: tr.start.valueOf(),
          end: tr.end.valueOf(),
        },
      };

      console.log(taskConfig);
      const embeddedRack = await client.racks.retrieve({ name: "Node 1" });
      await embeddedRack.createTask({
        name: `Export to Sift: ${runName}`,
        type: "sift_upload",
        config: taskConfig,
      });

      addStatus({
        variant: "success",
        message: `Export to Sift started for ${runName}`,
      });

      handleFinish();
    } catch (e) {
      addStatus({
        variant: "error",
        message: `Failed to start export: ${String(e)}`,
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
      Export to Sift
    </Button.Button>
  );
};

export const [useExportModal, ExportModal] = Modals.createBase<void, ExportModalArgs>(
  "Export.Sift",
  EXPORT_MODAL_LAYOUT_TYPE,
  ({ value: { timeRange, channels, name }, onFinish }) => {
    const tr = new TimeRange(timeRange);
    const form = Form.use<typeof formSchema>({
      schema: formSchema,
      values: {
        name,
        deviceKey: "",
        assetName: "",
        flowName: "telemetry",
        runName: name,
        channels,
        timeRange: tr.numeric,
      },
    });

    const footer = (
      <>
        <Triggers.SaveHelpText action="Export" />
        <Nav.Bar.End x align="center">
          <ExportButton handleFinish={onFinish} timeRange={timeRange} />
        </Nav.Bar.End>
      </>
    );

    return (
      <Form.Form<typeof formSchema> {...form}>
        <Modals.ModalContentLayout footer={footer} gap="huge">
          <Flex.Box y gap="small">
            <Text.Text level="h3" weight={450}>
              Export {name} to Sift
            </Text.Text>
            <Text.Text level="small" color={8}>
              Export telemetry data to Sift for analysis and visualization.
            </Text.Text>
          </Flex.Box>
          <Flex.Box y full="x" gap="medium">
            <Form.Field<string> path="deviceKey" label="Sift Device" required>
              {({ value, onChange }) => (
                <SiftDeviceSelect value={value} onChange={onChange} />
              )}
            </Form.Field>
            <Flex.Box x gap="medium">
              <Form.Field<string> path="assetName" label="Asset Name" required grow>
                {(p) => <Input.Text placeholder="My Asset" {...p} />}
              </Form.Field>
              <Form.Field<string> path="flowName" label="Flow Name" required grow>
                {(p) => <Input.Text placeholder="telemetry" {...p} />}
              </Form.Field>
            </Flex.Box>
            <Form.Field<string> path="runName" label="Run Name" required>
              {(p) => <Input.Text placeholder="Test Run 1" {...p} />}
            </Form.Field>
            <Form.Field<channel.Keys> path="channels" label="Channels" required>
              {({ value, onChange }) => (
                <Channel.SelectMultiple
                  value={value}
                  onChange={onChange}
                  full="x"
                  triggerProps={{ placeholder: "Select channels to export" }}
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
