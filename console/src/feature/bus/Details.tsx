// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type bus, type channel, type library } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";
import { Channel, Device } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode } from "react";

import { type DetailsProps } from "@/feature/bus/Messages";
import { commandChannelName, fieldChannelName, indexName } from "@/feature/bus/names";
import { CSS } from "@/platform/css";
import { Task } from "@/platform/task";

/** @returns the name of the device the form's config names, or "" until it loads. */
const useDeviceName = (): string => {
  const key = Form.useFieldValue<string>("config.device");
  const { data } = Device.useResult(primitive.isNonZero(key) ? { key } : null);
  return data?.name ?? "";
};

interface ChannelLabelProps {
  channel: channel.Key;
  /** The name configure gives the channel it creates. */
  planned: string;
}

/** Names a bound channel, or the channel the task creates when it starts. */
const ChannelLabel = ({ channel, planned }: ChannelLabelProps): ReactElement => {
  const { data: name } = Channel.useResultName(channel === 0 ? null : { key: channel });
  if (channel !== 0)
    return (
      <Text.Text level="small" color={10} overflow="ellipsis">
        {name ?? ""}
      </Text.Text>
    );
  return (
    <Text.Text level="small" color={8} overflow="ellipsis">
      {planned === "" ? "Select a device" : `Creates ${planned}`}
    </Text.Text>
  );
};

// Read and write tasks bind a field to a channel the same way.
type Bound = bus.ReadField & bus.WriteField;

interface FieldListProps {
  entry: library.MessageEntry;
  value: Bound[];
  onChange: (value: Bound[]) => void;
  /** Renders how an included field binds to its channel. */
  binding: (
    field: library.Field,
    bound: Bound,
    update: (next: Bound) => void,
  ) => ReactNode;
}

/**
 * Checks the fields of a message a task binds, in library order. A newly checked field
 * has no channel until configure creates one.
 */
const FieldList = ({
  entry,
  value,
  onChange,
  binding,
}: FieldListProps): ReactElement => {
  const isPreview = Task.useIsPreview();
  const byField = new Map(value.map((f) => [f.field, f]));
  const toggle = (key: library.FieldKey, included: boolean) => {
    const next = new Map(byField);
    if (included) next.set(key, { field: key, channel: 0 });
    else next.delete(key);
    onChange(entry.fields.flatMap((f) => next.get(f.key) ?? []));
  };
  const update = (bound: Bound) =>
    onChange(value.map((f) => (f.field === bound.field ? bound : f)));
  return (
    <Flex.Box y gap="small" className={CSS.B("bus-fields")}>
      {entry.fields.map((f) => {
        const bound = byField.get(f.key);
        return (
          <Flex.Box key={f.key} x align="center" className={CSS.B("bus-field")}>
            <Input.Checkbox
              value={bound != null}
              onChange={(included) => toggle(f.key, included)}
              disabled={isPreview}
              aria-label={f.name}
            />
            <Flex.Box y gap="tiny" className={CSS.BE("bus-field", "name")}>
              <Text.Text weight={500}>{f.name}</Text.Text>
              {f.units !== "" && (
                <Text.Text level="small" color={8}>
                  {f.units}
                </Text.Text>
              )}
            </Flex.Box>
            {bound != null && binding(f, bound, update)}
          </Flex.Box>
        );
      })}
      {entry.fields.length === 0 && (
        <Text.Text status="disabled">The message has no fields</Text.Text>
      )}
    </Flex.Box>
  );
};

/** Shows the channels a read task writes a message's fields to. */
export const ReadDetails = ({ path, entry }: DetailsProps): ReactElement => {
  const device = useDeviceName();
  const index = Form.useFieldValue<channel.Key>(`${path}.index`);
  const plan = (name: string) => (device === "" ? "" : name);
  return (
    <Form.Sections className={CSS.B("bus-details")}>
      <Form.Section title="Timestamps">
        <ChannelLabel channel={index} planned={plan(indexName(device, entry.name))} />
      </Form.Section>
      <Form.Section title="Fields">
        <Form.Field<bus.ReadField[]> path={`${path}.fields`} showLabel={false}>
          {({ value, onChange }) => (
            <FieldList
              entry={entry}
              value={value}
              onChange={onChange}
              binding={(f, bound) => (
                <ChannelLabel
                  channel={bound.channel}
                  planned={plan(fieldChannelName(device, entry.name, f.name))}
                />
              )}
            />
          )}
        </Form.Field>
      </Form.Section>
    </Form.Sections>
  );
};

/** Maps the fields a write task sends to command channels. */
export const WriteDetails = ({ path, entry }: DetailsProps): ReactElement => {
  const device = useDeviceName();
  const isPreview = Task.useIsPreview();
  const period =
    entry.period == null ? "on change" : `every ${entry.period.toString()}`;
  return (
    <Form.Sections className={CSS.B("bus-details")}>
      <Form.Section title="Fields">
        <Text.Text level="small" color={8}>
          {`Sent ${period}. A field that is not checked is sent as zero.`}
        </Text.Text>
        <Form.Field<bus.WriteField[]> path={`${path}.fields`} showLabel={false}>
          {({ value, onChange }) => (
            <FieldList
              entry={entry}
              value={value}
              onChange={onChange}
              binding={(f, bound, update) => (
                <Flex.Box y gap="tiny" grow className={CSS.BE("bus-field", "binding")}>
                  <Channel.SelectSingle
                    value={bound.channel === 0 ? undefined : bound.channel}
                    onChange={(key: channel.Key | null) =>
                      update({ ...bound, channel: key ?? 0 })
                    }
                    allowNone
                    disabled={isPreview}
                  />
                  {bound.channel === 0 && device !== "" && (
                    <Text.Text level="small" color={8}>
                      {`Creates ${commandChannelName(device, entry.name, f.name)}`}
                    </Text.Text>
                  )}
                </Flex.Box>
              )}
            />
          )}
        </Form.Field>
      </Form.Section>
    </Form.Sections>
  );
};
