// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/framer/DownloadCSVModal.css";

import { channel, DisconnectedError } from "@synnaxlabs/client";
import {
  Button,
  Channel,
  Flex,
  Form,
  Icon,
  Input,
  Nav,
  type Select,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import {
  type CrudeTimeRange,
  numericTimeRangeZ,
  runtime,
  TimeRange,
} from "@synnaxlabs/x";
import { z } from "zod";

import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Runtime } from "@/platform/runtime";
import { Triggers } from "@/platform/triggers";

export interface DownloadCSVModalParams {
  channelNames?: Record<channel.Key, string>;
  timeRange: CrudeTimeRange;
  channels: channel.Key[];
  name: string;
  icon?: Icon.ReactElement;
}

const NON_VIRTUAL_CHANNEL_QUERY: Partial<Channel.RetrieveMultipleQuery> = {
  virtual: false,
};
const CHANNEL_SELECT_TRIGGER_PROPS: Select.MultipleTriggerProps<channel.Key> = {
  placeholder: "Select channels to download",
};

export interface PromptDownloadCSV extends Modals.Prompt<
  void,
  DownloadCSVModalParams
> {}

export const useDownloadCSVModal = Modals.createPrompt<void, DownloadCSVModalParams>(
  ({ timeRange, channels, name, channelNames, icon, close }) => {
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
          <DownloadButton handleFinish={close} />
        </Nav.Bar.End>
      </>
    );
    return (
      <Form.Form<typeof formSchema> {...form}>
        <Modals.Frame className={CSS.B("download-csv")}>
          <Modals.Header icon={icon}>CSV.Download</Modals.Header>
          <Modals.Body gap="huge">
            <Text.Text level="h3" weight={450}>
              Download data for {name} as CSV
            </Text.Text>
            <Flex.Box y full="x" gap="medium">
              <Flex.Box x gap="medium">
                <Form.Field<number>
                  path="timeRange.start"
                  padHelpText={false}
                  label="From"
                >
                  {(p) => (
                    <Input.DateTime level="h4" variant="text" onlyChangeOnBlur {...p} />
                  )}
                </Form.Field>
                <Icon.Arrow.Right
                  className={CSS.BE("download-csv", "arrow")}
                  color={9}
                />
                <Form.Field<number> padHelpText={false} path="timeRange.end" label="To">
                  {(p) => (
                    <Input.DateTime onlyChangeOnBlur level="h4" variant="text" {...p} />
                  )}
                </Form.Field>
              </Flex.Box>
              <Form.Field<channel.Key[]> path="channels">
                {({ value, onChange }) => (
                  <Channel.SelectMultiple
                    value={value}
                    onChange={onChange}
                    initialQuery={NON_VIRTUAL_CHANNEL_QUERY}
                    triggerProps={CHANNEL_SELECT_TRIGGER_PROPS}
                    full="x"
                  />
                )}
              </Form.Field>
              <DownsampleFactorField
                path="downsampleFactor"
                label="Downsample factor"
              />
              {runtime.getOS() !== "Windows" && (
                <Text.Text status="warning" weight={450}>
                  Large downloads are faster when the Console runs in Google Chrome or
                  Microsoft Edge.
                </Text.Text>
              )}
            </Flex.Box>
          </Modals.Body>
          <Modals.Footer>{footer}</Modals.Footer>
        </Modals.Frame>
      </Form.Form>
    );
  },
);

const DownsampleFactorField = Form.buildNumericField({
  inputProps: { className: CSS.BE("download-csv", "downsample-factor") },
});

interface DownloadButtonProps {
  handleFinish: () => void;
}

const DownloadButton = ({ handleFinish }: DownloadButtonProps) => {
  const handleError = Status.useErrorHandler();
  const client = Synnax.use();
  const download = Runtime.useDownload();
  const { get } = Form.useContext();
  const handleClick = () => {
    const timeRange = get<TimeRange>("timeRange").value;
    const channels = get<channel.Key[]>("channels").value;
    const downsampleFactor = get<number>("downsampleFactor").value;
    const channelNames = get<Record<channel.Key, string>>("channelNames", {
      optional: true,
    })?.value;
    const name = get<string>("name").value;
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      const stream = await client.read({
        timeRange,
        channels,
        channelNames,
        iteratorConfig: { downsampleFactor },
        responseType: "csv",
      });
      await download({
        stream,
        name,
        extension: "csv",
        onDownloadStart: handleFinish,
      });
    }, `Failed to download CSV data for ${name}`);
  };
  const channelKeys = Form.useFieldValue<channel.Key[]>("channels");
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
