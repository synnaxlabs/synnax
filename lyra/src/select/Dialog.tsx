// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/select/Dialog.css";

import { destructor } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactNode,
  useState,
  useSyncExternalStore,
} from "react";
import { createPortal } from "react-dom";

import { CSS } from "@/css";
import { Dialog as BaseDialog } from "@/dialog";
import { Input } from "@/input";
import { ClosedContext } from "@/select/scope";

/** Props for {@link Dialog}. */
export interface DialogProps extends Omit<BaseDialog.DialogProps, "passthrough"> {}

const subscribe = (): destructor.Destructor => destructor.NOOP;
const getSnapshot = (): boolean => true;
const getServerSnapshot = (): boolean => false;

const Closed = ({ children }: PropsWithChildren): ReactNode => {
  const [detached] = useState(() => document.createElement("div"));
  return createPortal(<ClosedContext value>{children}</ClosedContext>, detached);
};

/**
 * The floating surface of a selection. While closed it renders its children into an
 * element outside the page, so a selected fixed {@link Item} can still show its label
 * in the trigger.
 */
export const Dialog = ({ className, children, ...rest }: DialogProps): ReactNode => {
  const { visible } = BaseDialog.useContext();
  // False on the server and during hydration, since the server renders no portal.
  const hydrated = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
  const content = <Input.ItemBoundary>{children}</Input.ItemBoundary>;
  if (!visible) return hydrated && <Closed>{content}</Closed>;
  return (
    <BaseDialog.Dialog
      {...rest}
      className={CSS.cls(CSS.BE("select", "dialog"), className)}
      bordered={false}
    >
      {content}
    </BaseDialog.Dialog>
  );
};
