// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/select/Dialog.css";

import { type record } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode } from "react";

import { Dialog as Core } from "@/dialog";
import { List } from "@/list";
import { Provider } from "@/select/Provider";
import {
  type UseMultipleProps,
  type UseReturn,
  type UseSingleProps,
} from "@/select/use";

export interface TriggerProps<
  K extends record.Key,
  E extends record.Keyed<K> | undefined,
> {
  value: K | null;
  useItem: (key: K) => E;
  onClick: () => void;
}

export type DialogProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K>,
> = UseReturn<K> &
  Pick<UseSingleProps<K> | UseMultipleProps<K>, "value"> &
  Pick<List.ListProps<K, E>, "data" | "useItem"> &
  Omit<Core.DialogProps, "visible" | "close" | "open" | "onSelect">;

type OmittedDialogProps<K extends record.Key> = Omit<
  DialogProps<K>,
  | "value"
  | "onSelect"
  | "data"
  | "useItem"
  | "children"
  | "onChange"
  | "clear"
  | "hover"
>;

export interface MultipleProps<K extends record.Key = record.Key>
  extends Omit<UseMultipleProps<K>, "data">,
    OmittedDialogProps<K> {}

export interface SingleProps<K extends record.Key = record.Key>
  extends Omit<UseSingleProps<K>, "data">,
    OmittedDialogProps<K>,
    Pick<List.ItemsProps<K>, "emptyContent"> {
  disabled?: boolean;
  placeholder?: ReactNode;
}

export const Dialog = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K>,
>({
  onSelect,
  value,
  data,
  clear,
  useItem,
  children,
  hover,
  ...rest
}: DialogProps<K, E>): ReactElement => (
  <Provider value={value} onSelect={onSelect} clear={clear} hover={hover}>
    <List.List data={data} useItem={useItem}>
      <Core.Frame {...rest}>{children}</Core.Frame>
    </List.List>
  </Provider>
);
