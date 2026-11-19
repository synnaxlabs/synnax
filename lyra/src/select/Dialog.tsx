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
import { Menu } from "@/menu";
import { usePath } from "@/menu/filter";
import { ClosedContext, InlineContext, useViewingContext } from "@/select/scope";

/** Props for {@link Dialog}. */
export interface DialogProps extends BaseDialog.DialogProps {
  /**
   * The surface the dialog draws. A menu surface looks like a `Menu.Dialog`: its search
   * spans the top, and its options take the size of menu items. It does not filter as
   * the user types.
   */
  surface?: "dialog" | "menu";
  /**
   * The options that match the filter of the menu show in the menu, after the name of
   * the submenu, while the menu filters by text that the name does not match. Only a
   * select in a `Menu.Sub` uses it.
   */
  filtered?: boolean;
}

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
 * in the trigger. Its children can show a view in their place with `Dialog.useViews`,
 * except in a menu that shows the options inline.
 */
export const Dialog = ({
  ref,
  className,
  children,
  surface = "dialog",
  filtered = false,
  ...rest
}: DialogProps): ReactNode => {
  const { visible } = BaseDialog.useContext();
  // Under a submenu that the menu filter passes, a selection shows its options in the
  // menu or nothing.
  const nested = usePath().length > 0;
  const cls = CSS.cls(CSS.BE("select", "dialog"), className);
  const menuSurface = Menu.useSurface(ref, cls);
  // False on the server and during hydration, since the server renders no portal.
  const hydrated = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
  const setViewing = useViewingContext();
  const options = <Input.ItemBoundary>{children}</Input.ItemBoundary>;
  if (nested && filtered) return <InlineContext value>{options}</InlineContext>;
  const content = (
    <BaseDialog.Views onViewingChange={setViewing}>{options}</BaseDialog.Views>
  );
  if (!visible || nested) return hydrated && <Closed>{content}</Closed>;
  if (surface === "menu")
    return (
      <BaseDialog.Dialog {...menuSurface} {...rest}>
        {content}
      </BaseDialog.Dialog>
    );
  return (
    <BaseDialog.Dialog ref={ref} {...rest} className={cls} bordered={false}>
      {content}
    </BaseDialog.Dialog>
  );
};
