// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/framer/DeleteModal.css";

import { channel, DisconnectedError } from "@synnaxlabs/client";
import {
  Button,
  Channel,
  Component,
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
  type NumericTimeRange,
  numericTimeRangeZ,
  TimeRange,
  TimeStamp,
} from "@synnaxlabs/x";
import { type ReactElement, useCallback, useState } from "react";
import { z } from "zod";

import { CSS } from "@/css";
import { type Layout } from "@/layout";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

export const DELETE_LAYOUT: Layout.BaseState = {
  type: "delete_data",
  location: "modal",
  icon: "Channel",
  name: "Data.Delete",
  window: { resizable: false, size: { height: 350, width: 700 }, navTop: true },
};

const formSchema = z.object({
  channels: channel.keyZ.array().min(1, "Select at least one channel"),
  timeRange: numericTimeRangeZ.refine(({ start, end }) => start < end, {
    error: "Start time must be before end time",
    path: ["start"],
  }),
});

const CHANNEL_SELECT_TRIGGER_PROPS: Select.MultipleTriggerProps<channel.Key> = {
  placeholder: "Select channels to delete",
};

export const DeleteModal: Layout.Renderer = ({ onClose }) => {
  const [step, setStep] = useState<"form" | "confirm">("form");
  const methods = Form.use({
    schema: formSchema,
    values: { channels: [], timeRange: TimeRange.MAX.numeric },
  });
  return (
    <Form.Form<typeof formSchema> {...methods}>
      <Flex.Box align="stretch" direction="y" empty grow>
        {step === "form" ? (
          <FormStep onNext={() => setStep("confirm")} />
        ) : (
          <ConfirmStep onBack={() => setStep("form")} onClose={onClose} />
        )}
      </Flex.Box>
    </Form.Form>
  );
};

interface FormStepProps {
  onNext: () => void;
}

const FormStep = ({ onNext }: FormStepProps): ReactElement => {
  const { validate, set } = Form.useContext();
  const channelKeys = Form.useFieldValue<channel.Keys>("channels");
  const start = Form.useFieldValue<number>("timeRange.start");
  const end = Form.useFieldValue<number>("timeRange.end");
  const isFromBeginning = start === TimeRange.MAX.numeric.start;
  const isToEnd = end === TimeRange.MAX.numeric.end;
  const handleNext = useCallback(() => {
    if (validate()) onNext();
  }, [validate, onNext]);
  const footer = (
    <>
      <Triggers.SaveHelpText action="Next" />
      <Nav.Bar.End>
        <Button.Button
          variant="filled"
          disabled={channelKeys.length === 0}
          onClick={handleNext}
          trigger={Triggers.SAVE}
        >
          Next
        </Button.Button>
      </Nav.Bar.End>
    </>
  );
  return (
    <Modals.ModalContentLayout footer={footer} gap="large" justify="start">
      <Text.Text level="h3" weight={450}>
        Delete Data
      </Text.Text>
      <Flex.Box y full="x" gap="medium">
        <Form.Field<channel.Keys> path="channels">{channelSelectRenderProp}</Form.Field>
        <Flex.Box x gap="medium" align="start">
          <Flex.Box y gap="small" className={CSS.BE("delete-modal", "time-range-side")}>
            <Flex.Box x align="center" gap="small">
              <Input.Checkbox
                value={isFromBeginning}
                onChange={(v) =>
                  set(
                    "timeRange.start",
                    v ? TimeRange.MAX.numeric.start : TimeStamp.now().nanoseconds,
                  )
                }
              />
              <Text.Text weight={450}>From beginning of time</Text.Text>
            </Flex.Box>
            {!isFromBeginning && (
              <Form.Field<number>
                path="timeRange.start"
                padHelpText={false}
                label="From"
              >
                {inputDateTimeRenderProp}
              </Form.Field>
            )}
          </Flex.Box>
          <Icon.Arrow.Right className={CSS.BE("delete-modal", "arrow")} color={9} />
          <Flex.Box y gap="small" className={CSS.BE("delete-modal", "time-range-side")}>
            <Flex.Box x align="center" gap="small">
              <Input.Checkbox
                value={isToEnd}
                onChange={(v) =>
                  set(
                    "timeRange.end",
                    v ? TimeRange.MAX.numeric.end : TimeStamp.now().nanoseconds,
                  )
                }
              />
              <Text.Text weight={450}>To end of time</Text.Text>
            </Flex.Box>
            {!isToEnd && (
              <Form.Field<number> path="timeRange.end" padHelpText={false} label="To">
                {inputDateTimeRenderProp}
              </Form.Field>
            )}
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
    </Modals.ModalContentLayout>
  );
};

const channelSelectRenderProp = Component.renderProp(
  (p: Channel.SelectMultipleProps) => (
    <Channel.SelectMultiple
      triggerProps={CHANNEL_SELECT_TRIGGER_PROPS}
      full="x"
      {...p}
    />
  ),
);

const inputDateTimeRenderProp = Component.renderProp((p: Input.DateTimeProps) => (
  <Input.DateTime level="h4" variant="text" onlyChangeOnBlur {...p} />
));

const formatTimeRange = (start: number, end: number): string => {
  const startStr =
    start === TimeRange.MAX.start.nanoseconds
      ? "beginning of time"
      : new TimeStamp(start).toString("dateTime", "local");
  const endStr =
    end === TimeRange.MAX.end.nanoseconds
      ? "end of time"
      : new TimeStamp(end).toString("dateTime", "local");
  return `${startStr} to ${endStr}`;
};

interface ConfirmStepProps {
  onBack: () => void;
  onClose: () => void;
}

const newChannelStr = (channelKeys: channel.Keys): string =>
  channelKeys.length === 1 ? "one channel" : `${channelKeys.length} channels`;

const ConfirmStep = ({ onBack, onClose }: ConfirmStepProps): ReactElement => {
  const { get } = Form.useContext();
  const channelKeys = Form.useFieldValue<channel.Keys>("channels");
  const start = Form.useFieldValue<number>("timeRange.start");
  const end = Form.useFieldValue<number>("timeRange.end");
  const client = Synnax.use();
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  const handleDelete = useCallback(() => {
    const keys = get<channel.Keys>("channels").value;
    const tr = get<NumericTimeRange>("timeRange").value;
    onClose();
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      await client.delete(keys, tr);
      addStatus({
        variant: "success",
        message: `Successfully deleted data from ${newChannelStr(keys)}`,
      });
    }, "Failed to delete data");
  }, [get, onClose, handleError, client, addStatus]);

  const channelStr = newChannelStr(channelKeys);
  const footer = (
    <>
      <Nav.Bar.Start>
        <Button.Button variant="outlined" onClick={onBack}>
          Back
        </Button.Button>
      </Nav.Bar.Start>
      <Nav.Bar.End>
        <Button.Button
          variant="filled"
          status="error"
          onClick={handleDelete}
          onClickDelay={500}
        >
          <Icon.Delete />
          Delete
        </Button.Button>
      </Nav.Bar.End>
    </>
  );
  return (
    <Modals.ModalContentLayout footer={footer} gap="large">
      <Text.Text level="h3" weight={450}>
        Are you sure you want to delete this data?
      </Text.Text>
      <Flex.Box y gap="medium">
        <Text.Text weight={450}>
          This will permanently delete data from {channelStr} for the time range:{" "}
          {formatTimeRange(start, end)}.
        </Text.Text>
        <Text.Text weight={450} status="error">
          This action is irreversible.
        </Text.Text>
      </Flex.Box>
    </Modals.ModalContentLayout>
  );
};
