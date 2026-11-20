// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, icd } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { ICD as PICD } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import type { FC, ReactElement } from "react";

import { FLAGS } from "@/flags";
import { CSS } from "@/platform/css";
import type { MessagePlan } from "@/platform/icd/configure";
import { DEFAULT_DATA_TYPES } from "@/platform/icd/dataType";
import { FieldsStatus, JSONField } from "@/platform/icd/Fields";
import type { FormatForm } from "@/platform/icd/Formats";
import { Match } from "@/platform/icd/Match";
import { usePlan } from "@/platform/icd/Plans";
import {
  fieldsPath,
  hiddenDataTypes,
  isIndexable,
  jsonFields,
  type ReadSource,
  reindex,
  type TreeField,
} from "@/platform/icd/read";
import { Panel } from "@/platform/panel";
import { Task } from "@/platform/task";

interface OpenInEditorProps {
  icdKey: icd.Key;
}

const EditorButton = ({ icdKey }: OpenInEditorProps): ReactElement => {
  const openTab = Panel.useOpenTab();
  return (
    <Button.Button
      variant="text"
      size="small"
      tooltip="Open in the ICD editor"
      onClick={() => openTab({ variant: "resource", resource: icd.resourceID(icdKey) })}
    >
      <Icon.Library />
      Open in editor
    </Button.Button>
  );
};

const OpenInEditor: FC<OpenInEditorProps> = FLAGS.icds ? EditorButton : () => null;

export interface AddFieldProps {
  onClick: () => void;
}

export const AddField = ({ onClick }: AddFieldProps): ReactElement | null => {
  const isPreview = Task.useIsPreview();
  if (isPreview) return null;
  return (
    <Button.Button variant="text" size="small" onClick={onClick}>
      <Icon.Add />
      Add field
    </Button.Button>
  );
};

/**
 * @returns the channel and default name that the channel name of a field shows: the
 * name that deploying gives a channel it creates, or the field's own channel.
 */
const previewOf = (
  plan: MessagePlan | undefined,
  key: string,
  ch: channel.Key,
  defaultName: string,
): { channel: channel.Key; defaultName: string } => {
  const planned = plan == null ? undefined : Task.newChannelName(plan.channels, key);
  return planned == null
    ? { channel: ch, defaultName }
    : { channel: 0, defaultName: planned };
};

interface TimestampFieldsProps {
  path: deep.Path<ReadSource>;
  /** The index of the message in the source's reads. */
  read: number;
  /** The fields of the message format. */
  fields: icd.JSONField[];
  /** The prefix of the names of the channels that deploying creates. */
  prefix: string;
  /** The channel that a new index field reads into. */
  indexChannel: channel.Key;
  /** Names the time that the source stamps each sample with, such as "Arrival time". */
  sourceLabel: string;
  plan?: MessagePlan;
}

type TimingMode = "source" | "payload";

/** Picks where each sample of a message gets its time. */
const TimestampFields: FC<TimestampFieldsProps> = ({
  path,
  read,
  fields,
  prefix,
  indexChannel,
  sourceLabel,
  plan,
}) => {
  const { get, set } = PForm.useContext();
  const readPath = path.messages[read];
  const held = PForm.useFieldValue(readPath.index, { optional: true });
  const bindings = PForm.useFieldValue(readPath.bindings);
  const sourceDisabled = PForm.useFieldValue(path.disabled);
  const indexBinding = bindings.find(({ field }) => field === held);
  const timestamps = fields.filter(({ type }) => type === "timestamp");
  const indexable = timestamps.filter(isIndexable);
  const index = timestamps.find(({ key }) => key === held);
  const setIndex = (next: string | undefined) =>
    set(readPath, reindex(get(readPath).value, next, indexChannel));
  const handleMode = (mode: TimingMode) =>
    setIndex(mode === "payload" ? indexable.at(0)?.key : undefined);
  const invalid = index != null && !isIndexable(index);
  return (
    <>
      <Input.Item
        label="Source"
        helpText={
          timestamps.length === 0
            ? "Give a field a timestamp type to read the time from the payload."
            : indexable.length === 0
              ? "Remove the default of a timestamp field to read the time from the payload."
              : undefined
        }
      >
        <Select.Buttons<TimingMode>
          value={index == null ? "source" : "payload"}
          onChange={handleMode}
        >
          <Select.Item<TimingMode> itemKey="source">{sourceLabel}</Select.Item>
          <Select.Item<TimingMode> itemKey="payload" disabled={indexable.length === 0}>
            Payload value
          </Select.Item>
        </Select.Buttons>
      </Input.Item>
      {index != null && (
        <Input.Item
          label="Field"
          padHelpText={false}
          helpText={
            invalid
              ? `${index.name} has a default, so it cannot be the index. Remove its default in the message format.`
              : undefined
          }
          status={invalid ? "error" : undefined}
        >
          <Select.Simple<string>
            value={index.key}
            onChange={setIndex}
            resourceName="field"
            allowNone={false}
          >
            {timestamps.map((field) => (
              <Select.Item<string>
                key={field.key}
                itemKey={field.key}
                label={field.name}
                textValue={field.name}
                disabled={!isIndexable(field)}
              >
                <Icon.Time />
                {field.name}
                {!isIndexable(field) && (
                  <Select.ItemDescription>
                    It has a default, so it cannot be the index.
                  </Select.ItemDescription>
                )}
              </Select.Item>
            ))}
          </Select.Simple>
        </Input.Item>
      )}
      {indexBinding != null && (
        <Task.ChannelNameField
          {...previewOf(
            plan,
            indexBinding.field,
            indexBinding.channel,
            `${prefix}_time`,
          )}
          skipped={sourceDisabled}
        />
      )}
    </>
  );
};

export interface FieldDetailsProps {
  path: deep.Path<ReadSource>;
  /** The index of the message in the source's reads. */
  read: number;
  field: TreeField;
  format: FormatForm;
  /** The prefix of the names of the channels that deploying creates. */
  prefix: string;
}

/**
 * The format of a field, which each edit saves to the ICD at once, then the channel
 * that the task reads it into.
 */
export const FieldDetails: FC<FieldDetailsProps> = ({
  path,
  read,
  field,
  format,
  prefix,
}) => {
  const readPath = path.messages[read];
  const plan = usePlan(path, read);
  const bindings = PForm.useFieldValue(readPath.bindings);
  const message = PForm.useFieldValue(readPath.message);
  const icdKey = PForm.useFieldValue(path.icd);
  const sourceDisabled = PForm.useFieldValue(path.disabled);
  const i = bindings.findIndex(({ field: key }) => key === field.key);
  const preview =
    i < 0
      ? undefined
      : previewOf(
          plan,
          field.key,
          bindings[i].channel,
          `${prefix}_${channel.escapeInvalidName(field.name)}`,
        );
  return (
    <Flex.Box y grow empty className={CSS.BE("details", "form")}>
      <PForm.Sections>
        <PForm.Form<typeof PICD.formSchema> {...format}>
          <JSONField
            path={fieldsPath(message)[field.key]}
            actions={<OpenInEditor icdKey={icdKey} />}
          />
        </PForm.Form>
        <PForm.Section title="Channel">
          {preview == null ? (
            <Text.Text status="disabled">
              The task does not read this field. Click its dot to read it.
            </Text.Text>
          ) : (
            <>
              <Task.ChannelNameField {...preview} skipped={sourceDisabled} />
              <Task.Payload.DataTypeField
                path={readPath.bindings[i].dataType}
                channel={preview.channel}
                defaultDataType={DEFAULT_DATA_TYPES[field.type]}
                hideDataTypes={hiddenDataTypes(field.type)}
              />
            </>
          )}
        </PForm.Section>
      </PForm.Sections>
    </Flex.Box>
  );
};

export interface MessageSectionsProps extends Omit<
  TimestampFieldsProps,
  "fields" | "plan"
> {
  format: FormatForm;
  /** Says why the ICD has no message for the read, and what to do now. */
  missingDescription: string;
}

/** The timestamp of a message that a source reads, and the problems of its format. */
export const MessageSections: FC<MessageSectionsProps> = ({
  format,
  missingDescription,
  ...rest
}) => {
  const { path, read } = rest;
  const plan = usePlan(path, read);
  const icdKey = PForm.useFieldValue(path.icd);
  const readMessage = PForm.useFieldValue(path.messages[read]);
  const message = PForm.useFieldValue(PICD.formPath.messages[readMessage.message], {
    optional: true,
    ctx: format,
  });
  const fields = jsonFields(message);
  return (
    <>
      {message == null && (
        <PForm.Section title="Message">
          <Status.Summary
            variant="warning"
            message="Message not found"
            description={missingDescription}
            center={false}
          />
        </PForm.Section>
      )}
      {message != null && message.payload.format !== "json" && (
        <PForm.Section title="Message" actions={<OpenInEditor icdKey={icdKey} />}>
          <Text.Text color={9}>
            The format of this message is not JSON.
            {FLAGS.icds && " Open it to change its format."}
          </Text.Text>
        </PForm.Section>
      )}
      <PForm.Section title="Timestamp">
        <TimestampFields {...rest} fields={fields} plan={plan} />
      </PForm.Section>
      <PForm.Form<typeof PICD.formSchema> {...format}>
        <FieldsStatus path={fieldsPath(readMessage.message)} />
      </PForm.Form>
    </>
  );
};

export interface MessageDetailsProps extends MessageSectionsProps {
  /** Titles the section that matches the values of the message, such as "Documents". */
  matchTitle: string;
  /** Names the match option that takes every payload, such as "Any document". */
  anyLabel: string;
  onAddField: () => void;
}

/** The name and match of a message of a source that reads several. */
export const MessageDetails: FC<MessageDetailsProps> = ({
  matchTitle,
  anyLabel,
  onAddField,
  ...rest
}) => {
  const { path, read, format } = rest;
  const icdKey = PForm.useFieldValue(path.icd);
  const key = PForm.useFieldValue(path.messages[read].message);
  const encoding = PForm.useFieldValue(PICD.formPath.messages[key].payload.format, {
    optional: true,
    ctx: format,
  });
  return (
    <Flex.Box y grow empty className={CSS.BE("details", "form")}>
      <PForm.Sections>
        {encoding === "json" && (
          <PForm.Form<typeof PICD.formSchema> {...format}>
            <PForm.Section
              title="Message"
              actions={
                <>
                  <AddField onClick={onAddField} />
                  <OpenInEditor icdKey={icdKey} />
                </>
              }
            >
              <PForm.TextField
                path={PICD.formPath.messages[key].name}
                label="Name"
                padHelpText={false}
              />
            </PForm.Section>
            <PForm.Section title={matchTitle}>
              <Match
                path={deep.variant(
                  PICD.formPath.messages[key].payload,
                  "format",
                  "json",
                )}
                anyLabel={anyLabel}
              />
            </PForm.Section>
          </PForm.Form>
        )}
        <MessageSections {...rest} />
      </PForm.Sections>
    </Flex.Box>
  );
};
