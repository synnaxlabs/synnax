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
import type { Select } from "@synnaxlabs/lyra/select";
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
  /**
   * The time ranges to read, in any order. The modal can edit one range, and lists
   * several with no edit.
   */
  timeRanges: CrudeTimeRange[];
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

interface TimeRangeFieldsProps {
  index: number;
  preview: boolean;
  labelShown: boolean;
}

const TimeRangeFields = ({ index, preview, labelShown }: TimeRangeFieldsProps) => {
  const rangePath = path.timeRanges[index];
  const range = Form.useFieldValue(rangePath);
  return (
    <>
      <Form.DateTimeField
        path={rangePath.start}
        padHelpText={false}
        label="From"
        showLabel={labelShown}
        inputProps={{
          level: "h4",
          variant: "text",
          bound: "start",
          anchors: { end: range.end },
          preview,
        }}
      />
      <Icon.Arrow.Right className={CSS.BE("download-csv", "arrow")} color={9} />
      <Form.DateTimeField
        padHelpText={false}
        path={rangePath.end}
        label="To"
        showLabel={labelShown}
        inputProps={{
          level: "h4",
          variant: "text",
          bound: "end",
          anchors: { start: range.start },
          preview,
        }}
      />
    </>
  );
};

const DownloadCSV = ({
  timeRanges,
  channels,
  name,
  icon,
  close,
}: Modals.ContentProps<DownloadCSVModalParams, void>) => {
  // The list reads in the order of the rows in the file.
  const [given] = useState(() =>
    timeRanges.map((tr) => new TimeRange(tr)).sort((a, b) => TimeRange.sort(a, b)),
  );
  const [shown] = useState(() => given.map(shownRange));
  // Form times are floats that round nanoseconds, so a time left as shown reads the
  // exact time behind it. An open end left as shown stays open.
  const toRead = (ranges: NumericTimeRange[]): TimeRange[] =>
    ranges.map(
      ({ start, end }, i) =>
        new TimeRange(
          start === shown[i].numeric.start ? shown[i].start : start,
          end === shown[i].numeric.end ? given[i].end : end,
        ),
    );
  const form = Form.use<typeof formSchema>({
    schema: formSchema,
    values: {
      channels,
      timeRanges: shown.map((tr) => tr.numeric),
      downsampleFactor: 1,
      name,
    },
  });
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
            <div className={CSS.BE("download-csv", "spans")}>
              {shown.map((_, i) => (
                <TimeRangeFields
                  key={i}
                  index={i}
                  preview={shown.length > 1}
                  labelShown={i === 0}
                />
              ))}
            </div>
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
  /** @returns the ranges to read for the ranges in the form. */
  toRead: (ranges: NumericTimeRange[]) => TimeRange[];
}

const DownloadButton = ({ handleFinish, toRead }: DownloadButtonProps) => {
  const handleError = Status.useErrorHandler();
  const client = Synnax.use();
  const download = Runtime.useDownload();
  const { get } = Form.useContext();
  const handleClick = () => {
    const timeRanges = toRead(get(path.timeRanges).value);
    const channels = get(path.channels).value;
    const downsampleFactor = get(path.downsampleFactor).value;
    const name = get(path.name).value;
    handleError(
      async () => {
        if (client == null) throw new DisconnectedError();
        const stream = await client.readCSV(timeRanges, channels, {
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
  timeRanges: numericTimeRangeZ
    .refine(({ start, end }) => end >= start, {
      error: "End time must be after start time",
      path: ["end"],
    })
    .array(),
  downsampleFactor: z.int().min(1).default(1),
});
