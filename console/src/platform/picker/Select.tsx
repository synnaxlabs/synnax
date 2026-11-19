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
import { type Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select as Base } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Flux } from "@synnaxlabs/pluto";
import { type record, strings } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode } from "react";

import { CSS } from "@/platform/css";

type Entry<K extends record.Key> = record.KeyedNamed<K> & record.Unknown;

/** The line that describes an entry under its name. */
export interface Description {
  text: string;
  status?: "warning" | "error";
}

interface EntryProps<E> {
  icon: Icon.FC;
  describe: (entry: E) => Description;
}

interface ListItemProps<K extends record.Key, E>
  extends List.ItemRenderProps<K>, EntryProps<E> {}

const ListItem = <K extends record.Key, E extends Entry<K>>({
  icon: ItemIcon,
  describe,
  ...rest
}: ListItemProps<K, E>): ReactElement | null => {
  const entry = List.useItem<K, E>(rest.itemKey);
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
  preview?: boolean,
): string => {
  if (result.variant === "disabled")
    return preview === true ? "None" : `Select a ${resourceName}`;
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
  renderEndContent?: (entry: E) => ReactNode;
}

const Trigger = <K extends record.Key, E extends Entry<K>>({
  icon: ItemIcon,
  resourceName,
  result,
  preview,
  renderEndContent,
}: TriggerProps<E>): ReactElement => {
  const itemStatus = Input.useItemStatus({});
  const endContent =
    result.variant === "success" ? renderEndContent?.(result.data) : undefined;
  const content =
    result.variant === "success" ? (
      <Text.Text el="span" overflow="fade" className={CSS.BE("picker", "text")}>
        {result.data.name}
      </Text.Text>
    ) : (
      <Text.Text el="span" color={8} className={CSS.BE("picker", "text")}>
        {placeholderOf(result, resourceName, preview)}
      </Text.Text>
    );
  return (
    <Dialog.Trigger
      variant="outlined"
      gap="small"
      grow
      preview={preview}
      className={CSS.BE("picker", "trigger")}
      {...itemStatus}
    >
      <ItemIcon className={CSS.BE("picker", "icon")} />
      {content}
      {endContent != null && (
        <span className={CSS.BE("picker", "end")}>{endContent}</span>
      )}
    </Dialog.Trigger>
  );
};

export interface SelectProps<K extends record.Key, E extends Entry<K>>
  extends Input.Control<K>, EntryProps<E> {
  /** The singular name of what the select holds, such as "secret". */
  resourceName: string;
  /** The entries to pick from, from the domain's `useList`. */
  list: Flux.UseListReturn<K, E>;
  /** The held entry, from the domain's `useResult`. */
  result: Flux.Result<E>;
  /**
   * Opens a dialog that creates an entry with the search text as its name, which is
   * empty for an empty search. Resolves with its key, or null on cancel. Unset, the
   * list offers no create.
   */
  onCreate?: (name: string) => Promise<K | null>;
  preview?: boolean;
  /** Renders content from the held entry at the end of the trigger, such as a badge. */
  renderEndContent?: (entry: E) => ReactNode;
}

/**
 * Selects one stored entry by key. The trigger names the held entry live and says when
 * it is missing or unavailable. Each option of the list shows its description. The list
 * can end in an option that creates an entry.
 */
export const Select = <K extends record.Key, E extends Entry<K>>({
  value,
  onChange,
  resourceName,
  icon,
  describe,
  list,
  result,
  onCreate,
  preview,
  renderEndContent,
}: SelectProps<K, E>): ReactElement => {
  const { data, getItem, subscribe, status, more, search } = list;
  const handleCreate =
    onCreate == null
      ? undefined
      : (name: string) =>
          void onCreate(name).then((key) => {
            if (key != null) onChange(key);
          });
  return (
    <Dialog.Frame variant="connected" grow>
      <Base.Frame<K, E | undefined>
        data={data}
        getItem={getItem}
        subscribe={subscribe}
        onFetchMore={more}
        value={value}
        onChange={onChange}
        closeDialogOnSelect
      >
        <Trigger<K, E>
          icon={icon}
          resourceName={resourceName}
          result={result}
          preview={preview}
          renderEndContent={renderEndContent}
        />
        <Base.Dialog className={CSS.BE("picker", "dialog")}>
          <Base.Body<K>
            resourceName={resourceName}
            onSearch={search}
            status={status}
            footer={
              handleCreate != null && (
                <>
                  <Menu.Divider />
                  <Base.CreateItem<K, E>
                    resourceName={resourceName}
                    modal
                    onCreate={handleCreate}
                  />
                </>
              )
            }
          >
            {({ key, ...p }) => (
              <ListItem<K, E> key={key} {...p} icon={icon} describe={describe} />
            )}
          </Base.Body>
        </Base.Dialog>
      </Base.Frame>
    </Dialog.Frame>
  );
};
