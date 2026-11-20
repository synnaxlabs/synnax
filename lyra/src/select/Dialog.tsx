// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/select/Dialog.css";

import { array, destructor, type record } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactNode,
  useEffect,
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
import { useRegistryContext } from "@/select/registry";
import { ClosedContext, InlineContext } from "@/select/scope";

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

// Inputs that scroll the list or move its cursor. The first one ends a reveal that
// still waits, so the list never moves under the user.
const SCROLL_EVENTS = ["wheel", "touchmove", "pointerdown"] as const;
const SCROLL_KEYS: ReadonlySet<string> = new Set([
  "ArrowUp",
  "ArrowDown",
  "PageUp",
  "PageDown",
  "Home",
  "End",
]);

// The dialog opens on its first choice, as a native dropdown does. A fetched list can
// load the choice after the dialog is placed, so the reveal waits for it once. A
// virtual list adopts the scroll container of a dialog that opens again one render
// later, so the scroll waits a frame.
const useRevealChoice = (): void => {
  const { placed } = BaseDialog.useContext();
  const registry = useRegistryContext("Select.Dialog");
  const { scrollToIndex, getData, subscribeData } = List.useUtilContext();
  const getRowElement = List.useGetElement();
  const { getState } = useContext();
  // Read through a ref, so the reveal runs once per open as these change identity.
  const readRef = useSyncedRef({
    scrollToIndex,
    getData,
    subscribeData,
    getRowElement,
    getState,
  });
  useEffect(() => {
    if (!placed) return;
    const { scrollToIndex, getData, subscribeData, getRowElement, getState } =
      readRef.current;
    const selected = array.toArray(getState().value);
    let frame: number | undefined;
    const stops: destructor.Destructor[] = [];
    const stop = (): void => {
      stops.splice(0).forEach((s) => s());
      if (frame != null) cancelAnimationFrame(frame);
    };
    const reveal = (key: record.Key): void => {
      const element = registry.hasItem(key)
        ? registry.getElement(key)
        : getRowElement(key);
      if (element != null) element.scrollIntoView({ block: "nearest" });
      else scrollToIndex(getData().indexOf(key));
    };
    const tryReveal = (): void => {
      if (frame != null) return;
      const key = registry.getOrder(getData()).find((k) => selected.includes(k));
      if (key == null) return;
      frame = requestAnimationFrame(() => {
        frame = undefined;
        stop();
        reveal(key);
      });
    };
    const handleKey = (e: KeyboardEvent): void => {
      if (SCROLL_KEYS.has(e.key)) stop();
    };
    const options = { capture: true, passive: true };
    SCROLL_EVENTS.forEach((type) => window.addEventListener(type, stop, options));
    window.addEventListener("keydown", handleKey, options);
    stops.push(registry.subscribe(tryReveal), subscribeData(tryReveal));
    tryReveal();
    return () => {
      stop();
      SCROLL_EVENTS.forEach((type) => window.removeEventListener(type, stop, options));
      window.removeEventListener("keydown", handleKey, options);
    };
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
  const { visible } = BaseDialog.useContext();
  useRevealChoice();
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
