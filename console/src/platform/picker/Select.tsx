// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/picker/Select.css";

import { NotFoundError } from "@synnaxlabs/client";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select as Base } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Flux } from "@synnaxlabs/pluto";
import { primitive, type record, strings } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode } from "react";

import { CSS } from "@/platform/css";

type Entry = record.KeyedNamed<string> & record.Unknown;

// Entry keys are UUIDs, so no entry has this key.
const NONE = "none";

/** The line that describes an entry under its name. */
export interface Description {
  text: string;
  status?: "warning" | "error";
}

interface EntryProps<E> {
  icon: Icon.FC;
  describe: (entry: E) => Description;
}

interface ListItemProps<E> extends List.ItemRenderProps<string>, EntryProps<E> {}

const ListItem = <E extends Entry>({
  icon: ItemIcon,
  describe,
  ...rest
}: ListItemProps<E>): ReactElement | null => {
  const entry = List.useItem<string, E>(rest.itemKey);
  if (entry == null) return null;
  const { text, status } = describe(entry);
  return (
    <Base.Item {...rest} align="center" gap="medium">
      <ItemIcon className={CSS.BE("picker", "icon")} />
      <Flex.Box y empty grow className={CSS.BE("picker", "text")}>
        <Text.Text overflow="fade">{entry.name}</Text.Text>
        <Text.Text level="tiny" color={9} status={status} overflow="fade">
          {text}
        </Text.Text>
      </Flex.Box>
    </Base.Item>
  );
};

// The text of a trigger with no entry to name. It is empty while the entry loads.
const placeholderOf = <E extends record.Unknown>(
  result: Flux.Result<E>,
  resourceName: string,
  preview: boolean,
  allowNone: boolean,
): string => {
  if (result.variant === "disabled") {
    if (allowNone) return `No ${resourceName}`;
    return preview ? "None" : `Select a ${resourceName}`;
  }
  if (result.variant !== "error") return "";
  const { error } = result.status.details;
  return Flux.DeletedError.matches(error) || NotFoundError.matches(error.cause)
    ? `${strings.capitalize(resourceName)} not found`
    : `Unavailable ${resourceName}`;
};

interface TriggerProps<E extends record.Unknown> {
  icon: Icon.FC;
  resourceName: string;
  result: Flux.Result<E>;
  preview?: boolean;
  allowNone: boolean;
  renderEndContent?: (entry: E) => ReactNode;
}

const Trigger = <E extends Entry>({
  icon: ItemIcon,
  resourceName,
  result,
  preview,
  allowNone,
  renderEndContent,
}: TriggerProps<E>): ReactElement => {
  const itemStatus = Input.useItemStatus({});
  const clear = Base.useTriggerClear<string>({
    clearable: allowNone && result.variant !== "disabled",
    preview,
  });
  const icon = <ItemIcon className={CSS.BE("picker", "icon")} />;
  const endContent =
    result.variant === "success" ? renderEndContent?.(result.data) : undefined;
  const content =
    result.variant === "success" ? (
      <Text.Text el="span" overflow="fade" className={CSS.BE("picker", "text")}>
        {result.data.name}
      </Text.Text>
    ) : (
      <Text.Text el="span" color={8} className={CSS.BE("picker", "text")}>
        {placeholderOf(result, resourceName, preview === true, allowNone)}
      </Text.Text>
    );
  return (
    <Dialog.Trigger
      variant="outlined"
      gap="small"
      grow
      preview={preview}
      className={CSS.cls(
        CSS.BE("picker", "trigger"),
        clear.canClear && clear.className,
      )}
      onKeyDown={clear.handleKeyDown}
      {...itemStatus}
    >
      {clear.canClear ? (
        <Base.TriggerClear
          name={result.variant === "success" ? result.data.name : undefined}
        >
          {icon}
        </Base.TriggerClear>
      ) : (
        icon
      )}
      {content}
      {endContent != null && (
        <span className={CSS.BE("picker", "end")}>{endContent}</span>
      )}
    </Dialog.Trigger>
  );
};

interface RequiredProps extends Input.Control<string> {
  allowNone?: false;
}

interface AllowNoneProps {
  value: string;
  /** Gets null when the user clears the value. */
  onChange: (value: string | null) => void;
  /**
   * Whether the value can be empty. The list pins a "No <resourceName>" option, a
   * button covers the trigger icon on hover, and Delete or Backspace clears it.
   */
  allowNone: true;
}

interface BaseSelectProps<E extends Entry> extends EntryProps<E> {
  /** The singular name of what the select holds, such as "secret". */
  resourceName: string;
  /** The entries to pick from, from the domain's `useList`. */
  list: Flux.UseListReturn<string, E>;
  /** The held entry, from the domain's `useResult`. */
  result: Flux.Result<E>;
  /**
   * Opens a dialog that creates an entry with the search text as its name, which is
   * empty for an empty search. Resolves with its key, or null on cancel. Unset, the
   * list offers no create.
   */
  onCreate?: (name: string) => Promise<string | null>;
  preview?: boolean;
  /** Renders content from the held entry at the end of the trigger, such as a badge. */
  renderEndContent?: (entry: E) => ReactNode;
}

export type SelectProps<E extends Entry> = BaseSelectProps<E> &
  (RequiredProps | AllowNoneProps);

/**
 * Selects one stored entry by key. The trigger names the held entry live and says when
 * it is missing or unavailable. Each option of the list shows its description. The list
 * can end in an option that creates an entry.
 */
export const Select = <E extends Entry>(props: SelectProps<E>): ReactElement => {
  const {
    resourceName,
    icon,
    describe,
    list,
    result,
    onCreate,
    preview,
    renderEndContent,
  } = props;
  const { data, getItem, subscribe, status, more, search } = list;
  const selection =
    props.allowNone === true
      ? {
          value: primitive.isZero(props.value) ? NONE : props.value,
          onChange: (key: string | null) => props.onChange(key === NONE ? null : key),
          allowNone: true as const,
        }
      : { value: props.value, onChange: props.onChange, allowNone: false as const };
  const handleCreate =
    onCreate == null
      ? undefined
      : (name: string) =>
          void onCreate(name).then((key) => {
            if (key != null) props.onChange(key);
          });
  return (
    <Dialog.Frame variant="connected" grow>
      <Base.Frame<string, E | undefined>
        data={data}
        getItem={getItem}
        subscribe={subscribe}
        onFetchMore={more}
        {...selection}
        closeDialogOnSelect
      >
        <Trigger<E>
          icon={icon}
          resourceName={resourceName}
          result={result}
          preview={preview}
          allowNone={selection.allowNone}
          renderEndContent={renderEndContent}
        />
        <Base.Dialog className={CSS.BE("picker", "dialog")}>
          <Base.Body<string>
            resourceName={resourceName}
            onSearch={search}
            status={status}
            footer={
              (handleCreate != null || selection.allowNone) && (
                <>
                  <Menu.Divider />
                  {handleCreate != null && (
                    <Base.CreateItem<string, E>
                      resourceName={resourceName}
                      modal
                      onCreate={handleCreate}
                    />
                  )}
                  {selection.allowNone && (
                    <Base.Item<string> itemKey={NONE} textValue={`No ${resourceName}`}>
                      <Icon.None />
                      No {resourceName}
                    </Base.Item>
                  )}
                </>
              )
            }
          >
            {({ key, ...p }) => (
              <ListItem<E> key={key} {...p} icon={icon} describe={describe} />
            )}
          </Base.Body>
        </Base.Dialog>
      </Base.Frame>
    </Dialog.Frame>
  );
};
