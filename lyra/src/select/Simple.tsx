// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv, type record } from "@synnaxlabs/x";
import { plural } from "pluralize";
import { type ReactElement, type ReactNode } from "react";

import { Dialog } from "@/dialog";
import { DefaultEmptyContent } from "@/select/Body";
import { Dialog as SelectDialog } from "@/select/Dialog";
import { Frame, type MultipleFrameProps, type SingleFrameProps } from "@/select/Frame";
import { List } from "@/select/List";
import { MultipleTrigger, type MultipleTriggerProps } from "@/select/MultipleTrigger";
import { usePlaceholder } from "@/select/placeholder";
import { useVisibleCount } from "@/select/registry";
import { useClosed } from "@/select/scope";
import { Search } from "@/select/Search";
import { SingleTrigger, type SingleTriggerProps } from "@/select/SingleTrigger";

type FrameOmitted =
  | "multiple"
  | "children"
  | "data"
  | "getItem"
  | "subscribe"
  | "virtual"
  | "itemHeight"
  | "overscan"
  | "onFetchMore";

interface BaseSimpleProps
  extends
    Omit<Dialog.FrameProps, "onChange" | "children" | "variant">,
    Pick<SingleTriggerProps, "disabled" | "icon" | "haulType"> {
  /** Singular name of the thing being selected. It builds the placeholder and the
   * empty content. */
  resourceName: string;
  /** The options, as {@link Item}s. */
  children: ReactNode;
  /** Shown when the search hides every option. */
  emptyContent?: ReactNode;
  variant?: Dialog.FrameProps["variant"];
  /** Whether to render the trigger flat and inert, for use inside a preview. */
  preview?: boolean;
  dialogProps?: Dialog.FrameProps;
}

/** Props for a {@link Simple} that selects one option. */
export interface SingleSimpleProps<K extends record.Key>
  extends BaseSimpleProps, Omit<SingleFrameProps<K, undefined>, FrameOmitted> {
  multiple?: false;
  triggerProps?: SingleTriggerProps;
}

/** Props for a {@link Simple} that selects any number of options, shown as tags. */
export interface MultipleSimpleProps<K extends record.Key>
  extends
    BaseSimpleProps,
    Omit<MultipleFrameProps<K, undefined>, FrameOmitted | "replaceOnSingle"> {
  multiple: true;
  triggerProps?: MultipleTriggerProps<K, undefined>;
}

/** Props for {@link Simple}. Set `multiple` to select any number of options. */
export type SimpleProps<K extends record.Key> =
  SingleSimpleProps<K> | MultipleSimpleProps<K>;

const Empty = ({
  resourceName,
  emptyContent,
}: Pick<BaseSimpleProps, "resourceName" | "emptyContent">): ReactNode => {
  const closed = useClosed();
  const count = useVisibleCount();
  if (closed || count > 0) return null;
  return emptyContent ?? <DefaultEmptyContent resourceName={resourceName} />;
};

/**
 * A dropdown that selects one of a few fixed options, given as {@link Item} children.
 * With `multiple`, it selects any number of them and shows each as a tag. Its search
 * filters the options by their text.
 *
 * @example
 * <Select.Simple resourceName="mode" value={mode} onChange={setMode}>
 *   <Select.Item itemKey="fast">Fast</Select.Item>
 *   <Select.Item itemKey="slow">Slow</Select.Item>
 * </Select.Simple>
 */
export const Simple = <K extends record.Key>({
  resourceName,
  value,
  onChange,
  multiple,
  allowNone,
  autoSelectOnNone,
  initialHover,
  enableTriggers,
  closeDialogOnSelect = multiple !== true,
  children,
  emptyContent,
  haulType,
  disabled,
  icon,
  variant = "connected",
  preview,
  triggerProps,
  dialogProps,
  ...rest
}: SimpleProps<K>): ReactElement => {
  const placeholder = usePlaceholder(
    multiple === true ? plural(resourceName) : resourceName,
  );
  const triggerBase = {
    haulType,
    icon,
    placeholder,
    disabled,
    preview,
    "aria-label": caseconv.capitalize(
      multiple === true ? plural(resourceName) : resourceName,
    ),
  };
  const selection = {
    allowNone,
    autoSelectOnNone,
    initialHover,
    enableTriggers,
    closeDialogOnSelect,
  };
  const dialog = (
    <SelectDialog {...dialogProps}>
      <Search placeholder={`Search ${plural(resourceName)}...`} />
      <List bordered borderColor={6} grow rounded full="x">
        {children}
        <Empty resourceName={resourceName} emptyContent={emptyContent} />
      </List>
    </SelectDialog>
  );
  return (
    <Dialog.Frame {...rest} variant={variant}>
      {multiple === true ? (
        <Frame<K, undefined> multiple value={value} onChange={onChange} {...selection}>
          <MultipleTrigger<K, undefined> {...triggerBase} {...triggerProps} />
          {dialog}
        </Frame>
      ) : (
        <Frame<K, undefined> value={value} onChange={onChange} {...selection}>
          <SingleTrigger {...triggerBase} {...triggerProps} />
          {dialog}
        </Frame>
      )}
    </Dialog.Frame>
  );
};
