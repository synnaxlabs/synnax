// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/select/Dialog.css";

import { array, destructor } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactNode,
  useLayoutEffect,
  useState,
  useSyncExternalStore,
} from "react";
import { createPortal } from "react-dom";

import { CSS } from "@/css";
import { Dialog as BaseDialog } from "@/dialog";
import { useSyncedRef } from "@/hooks/ref";
import { Input } from "@/input";
import { List } from "@/list";
import { Menu } from "@/menu";
import { usePath } from "@/menu/filter";
import { useContext } from "@/select/Context";
import { onFirstInput } from "@/select/firstInput";
import { useRegistryContext } from "@/select/registry";
import { ClosedContext, InlineContext } from "@/select/scope";
import { useOrder } from "@/select/use";

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

// The dialog opens on its first choice, as a native dropdown does. A fetched list can
// load the choice after the dialog is placed, so the reveal waits for it once, until
// the user's first input. The list never moves under the user.
const useRevealChoice = (placed: boolean): void => {
  const registry = useRegistryContext("Select.Dialog");
  const order = useOrder();
  const { scrollToIndex, getData } = List.useUtilContext();
  const { getState } = useContext();
  // Read through a ref, so the reveal runs once per open as these change identity.
  const readRef = useSyncedRef({ order, scrollToIndex, getData, getState });
  useLayoutEffect(() => {
    if (!placed) return;
    const { order, scrollToIndex, getData, getState } = readRef.current;
    const selected = array.toArray(getState().value);
    const stops: destructor.Destructor[] = [];
    const stop = (): void => stops.splice(0).forEach((s) => s());
    const tryReveal = (): void => {
      // A search lists its matches in their own order.
      if (registry.getTerm() !== "") return stop();
      const key = order.getOrder().find((k) => selected.includes(k));
      if (key == null) return;
      stop();
      const element = order.getElement(key);
      if (element != null) element.scrollIntoView({ block: "nearest" });
      else scrollToIndex(getData().indexOf(key));
    };
    stops.push(onFirstInput(registry, stop), order.subscribe(tryReveal));
    tryReveal();
    return stop;
  }, [placed, registry, readRef]);
};

/**
 * The floating surface of a selection. While closed it renders its children into an
 * element outside the page, so a selected fixed {@link Item} can still show its label
 * in the trigger.
 */
export const Dialog = ({
  ref,
  className,
  children,
  surface = "dialog",
  filtered = false,
  ...rest
}: DialogProps): ReactNode => {
  const { visible, placed } = BaseDialog.useContext();
  useRevealChoice(placed);
  // Under a submenu that the menu filter passes, a selection shows its options in the
  // menu or nothing.
  const nested = usePath().length > 0;
  const cls = CSS.cls(CSS.BE("select", "dialog"), className);
  const menuSurface = Menu.useSurface(ref, cls);
  // False on the server and during hydration, since the server renders no portal.
  const hydrated = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
  const content = <Input.ItemBoundary>{children}</Input.ItemBoundary>;
  if (nested && filtered) return <InlineContext value>{content}</InlineContext>;
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
