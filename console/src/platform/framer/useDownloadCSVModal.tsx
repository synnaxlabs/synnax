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
import {
  type CrudeTimeRange,
  deep,
  type NumericTimeRange,
  numericTimeRangeZ,
  runtime,
  TimeRange,
  TimeStamp,
} from "@synnaxlabs/x";
import { useState } from "react";
import { z } from "zod";

import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Runtime } from "@/platform/runtime";
import { Triggers } from "@/platform/triggers";

const path = deep.path<z.infer<typeof formSchema>>();

export interface DownloadCSVModalParams {
  timeRange: CrudeTimeRange;
  channels: channel.Key[];
  name: string;
  icon?: Icon.ReactElement;
}

const NON_VIRTUAL_CHANNEL_QUERY: Channel.ListQuery = { where: { virtual: false } };
const CHANNEL_SELECT_TRIGGER_PROPS: Select.MultipleTriggerProps<channel.Key> = {
  placeholder: "Select channels to download",
};

// A range that has not started or still runs has no time set at that end, so the
// modal shows now there.
const shownRange = ({ start, end }: TimeRange): TimeRange => {
  const now = TimeStamp.now();
  const from = start.afterEq(TimeStamp.MAX) ? now : start;
  const to = end.afterEq(TimeStamp.MAX) ? TimeStamp.max(from, now) : end;
  return new TimeRange(from, to);
};

const DownloadCSV = ({
  timeRange,
  channels,
  name,
  icon,
  close,
}: Modals.ContentProps<DownloadCSVModalParams, void>) => {
  const [shown] = useState(() => shownRange(new TimeRange(timeRange)));
  // Form times are floats that round nanoseconds, so a time left as shown reads the
  // exact time behind it. An open end left as shown stays open.
  const toRead = ({ start, end }: NumericTimeRange): TimeRange =>
    new TimeRange(
      start === shown.numeric.start ? shown.start : start,
      end === shown.numeric.end ? new TimeRange(timeRange).end : end,
    );
  const form = Form.use<typeof formSchema>({
    schema: formSchema,
    values: {
      channels,
      timeRange: shown.numeric,
      downsampleFactor: 1,
      name,
    },
  });
  const range = Form.useFieldValue(path.timeRange, { ctx: form });
  const footer = (
    <Nav.Bar.End x align="center">
      <DownloadButton handleFinish={close} toRead={toRead} />
    </Nav.Bar.End>
  );
  return (
    <Form.Form<typeof formSchema> {...form}>
      <Modal.Frame className={CSS.B("download-csv")}>
        <Modal.Header icon={icon}>CSV.Download</Modal.Header>
        <Modal.Body gap="huge">
          <Text.Text level="h3" weight={450}>
            Download data for {name} as CSV
          </Text.Text>
          <Flex.Box y full="x" gap="medium">
            <Flex.Box x gap="medium">
              <Form.DateTimeField
                path={path.timeRange.start}
                padHelpText={false}
                label="From"
                inputProps={{
                  level: "h4",
                  variant: "text",
                  bound: "start",
                  anchors: { end: range.end },
                }}
              />
              <Icon.Arrow.Right className={CSS.BE("download-csv", "arrow")} color={9} />
              <Form.DateTimeField
                padHelpText={false}
                path={path.timeRange.end}
                label="To"
                inputProps={{
                  level: "h4",
                  variant: "text",
                  bound: "end",
                  anchors: { start: range.start },
                }}
              />
            </Flex.Box>
            <Form.Field path={path.channels}>
              {({ value, onChange }) => (
                <Channel.SelectMultiple
                  value={value}
                  onChange={onChange}
                  query={NON_VIRTUAL_CHANNEL_QUERY}
                  triggerProps={CHANNEL_SELECT_TRIGGER_PROPS}
                  full="x"
                />
              )}
            </Form.Field>
            <DownsampleFactorField
              path={path.downsampleFactor}
              label="Downsample factor"
            />
            {runtime.getOS() !== "Windows" && (
              <Text.Text status="warning" weight={450}>
                Large downloads are faster when the Console runs in Google Chrome or
                Microsoft Edge.
              </Text.Text>
            )}
          </Flex.Box>
        </Modal.Body>
        <Modal.Footer>{footer}</Modal.Footer>
      </Modal.Frame>
    </Form.Form>
  );
};

export const useDownloadCSVModal = Modals.create<DownloadCSVModalParams>(DownloadCSV);

const DownsampleFactorField = Form.buildNumericField({
  inputProps: { className: CSS.BE("download-csv", "downsample-factor") },
});

interface DownloadButtonProps {
  handleFinish: () => void;
  /** @returns the range to read for the range in the form. */
  toRead: (range: NumericTimeRange) => TimeRange;
}

const DownloadButton = ({ handleFinish, toRead }: DownloadButtonProps) => {
  const handleError = Status.useErrorHandler();
  const client = Synnax.use();
  const download = Runtime.useDownload();
  const { get } = Form.useContext();
  const handleClick = () => {
    const timeRange = toRead(get(path.timeRange).value);
    const channels = get(path.channels).value;
    const downsampleFactor = get(path.downsampleFactor).value;
    const name = get(path.name).value;
    handleError(
      async () => {
        if (client == null) throw new DisconnectedError();
        const stream = await client.readCSV(timeRange, channels, {
          reduction: { variant: "stride", factor: downsampleFactor },
        });
        await download({
          stream,
          name,
          extension: "csv",
          onDownloadStart: handleFinish,
        });
      },
      { message: "Failed to download CSV data", name },
    );
  };
  const channelKeys = Form.useFieldValue(path.channels);
  const isDisabled = channelKeys.length === 0;
  return (
    <Button.Button
      variant="filled"
      disabled={isDisabled}
      onClick={handleClick}
      trigger={Triggers.SAVE}
      triggerIndicator
    >
      <Icon.Download />
      Download
    </Button.Button>
  );
};

const formSchema = z.object({
  name: z.string(),
  channels: channel.keyZ.array(),
  timeRange: numericTimeRangeZ.refine(({ start, end }) => end >= start, {
    error: "End time must be after start time",
    path: ["end"],
  }),
  downsampleFactor: z.int().min(1).default(1),
});
