// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import {
  Button,
  Channel,
  Flex,
  Form,
  Icon,
  Input,
  Nav,
  type Select,
  Text,
} from "@synnaxlabs/pluto";
import { type CrudeTimeRange, numericTimeRangeZ, TimeRange } from "@synnaxlabs/x";
import { z } from "zod";

import { useDownload } from "@/csv/useDownload";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

export interface DownloadModalArgs extends Modals.BaseArgs<void> {
  channelNames?: Record<channel.Key, string>;
  timeRange: CrudeTimeRange;
  channels: channel.Keys;
  name: string;
}

export const DOWNLOAD_MODAL_LAYOUT_TYPE = "downloadCSV";
const NON_VIRTUAL_CHANNEL_QUERY: Partial<Channel.RetrieveMultipleQuery> = {
  virtual: false,
};
const CHANNEL_SELECT_TRIGGER_PROPS: Select.MultipleTriggerProps<channel.Key> = {
  placeholder: "Select channels to download",
};

export interface PromptDownload extends Modals.Prompt<void, DownloadModalArgs> {}

export const [useDownloadModal, DownloadModal] = Modals.createBase<
  void,
  DownloadModalArgs
>(
  "Download.CSV",
  DOWNLOAD_MODAL_LAYOUT_TYPE,
  ({ value: { timeRange, channels, name, channelNames }, onFinish }) => {
    const form = Form.use<typeof formSchema>({
      schema: formSchema,
      values: {
        channels,
        timeRange: new TimeRange(timeRange).numeric,
        downsampleFactor: 1,
        name,
        channelNames,
      },
    });
    const footer = (
      <>
        <Triggers.SaveHelpText action="Download" />
        <Nav.Bar.End x align="center">
          <DownloadButton handleFinish={onFinish} />
        </Nav.Bar.End>
      </>
    );
    return (
      <Form.Form<typeof formSchema> {...form}>
        <Modals.ModalContentLayout footer={footer} gap="huge">
          <Text.Text level="h3" weight={450}>
            Download data for {name} to a CSV
          </Text.Text>
          <Flex.Box y full="x" gap="large">
            <Flex.Box x gap="medium" align="center">
              <Form.Field<number>
                path="timeRange.start"
                padHelpText={false}
                label="From"
              >
                {(p) => (
                  <Input.DateTime level="h4" variant="text" onlyChangeOnBlur {...p} />
                )}
              </Form.Field>
              <Icon.Arrow.Right style={{ width: "3rem", height: "3rem" }} color={9} />
              <Form.Field<number> padHelpText={false} path="timeRange.end" label="To">
                {(p) => (
                  <Input.DateTime onlyChangeOnBlur level="h4" variant="text" {...p} />
                )}
              </Form.Field>
            </Flex.Box>
            <ChannelsField />
            <Form.NumericField
              path="downsampleFactor"
              label="Downsample Factor"
              inputProps={{ style: { width: "15rem" } }}
            />
            <Text.Text status="warning" weight={450}>
              For improved performance when downloading large datasets, we recommend
              exporting from the Console when it is running in Google Chrome or
              Microsoft Edge.
            </Text.Text>
          </Flex.Box>
        </Modals.ModalContentLayout>
      </Form.Form>
    );
  },
  { window: { resizable: false, size: { height: 475, width: 700 }, navTop: true } },
);

interface DownloadButtonProps {
  handleFinish: () => void;
}

const DownloadButton = ({ handleFinish }: DownloadButtonProps) => {
  const downloadCSV = useDownload();
  const { get } = Form.useContext();
  const handleClick = () => {
    const timeRange = get<TimeRange>("timeRange").value;
    const channels = get<channel.Keys>("channels").value;
    const downsampleFactor = get<number>("downsampleFactor").value;
    const channelNames = get<Record<channel.Key, string>>("channelNames", {
      optional: true,
    })?.value;
    const name = get<string>("name").value;
    downloadCSV({
      timeRange,
      channels,
      channelNames,
      iteratorConfig: { downsampleFactor },
      name,
      onDownloadStart: handleFinish,
    });
  };
  const channelKeys = Form.useFieldValue<channel.Keys>("channels");
  const isDisabled = channelKeys.length === 0;
  return (
    <Button.Button
      variant="filled"
      disabled={isDisabled}
      onClick={handleClick}
      trigger={Triggers.SAVE}
    >
      <Icon.Download />
      Download
    </Button.Button>
  );
};

const ChannelsField = () => {
  const { onChange } = Form.useField<channel.Keys>("channels");
  const currentKeys = Form.useFieldValue<channel.Keys>("channels");
  return (
    <Channel.SelectMultiple
      value={currentKeys}
      onChange={onChange}
      initialQuery={NON_VIRTUAL_CHANNEL_QUERY}
      triggerProps={CHANNEL_SELECT_TRIGGER_PROPS}
      full="x"
    />
  );
};

const formSchema = z.object({
  name: z.string(),
  channelNames: z.record(channel.keyZ, z.string()).optional(),
  channels: channel.keyZ.array(),
  timeRange: numericTimeRangeZ.refine(({ start, end }) => end >= start, {
    error: "End time must be after start time",
    path: ["end"],
  }),
  downsampleFactor: z.int().min(1).default(1),
});
