// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/select/SingleTrigger.css";

import { primitive, type record } from "@synnaxlabs/x";
import { type KeyboardEvent, type ReactElement, useId } from "react";

import { Button } from "@/button";
import { Caret } from "@/caret";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Haul } from "@/haul";
import { type Icon } from "@/icon";
import { Input } from "@/input";
import { List } from "@/list";
import { useContext, useSelected } from "@/select/Context";
import { Label } from "@/select/Label";
import { staticCanDrop } from "@/select/MultipleTrigger";
import { useIsFixed, useText } from "@/select/registry";

export interface SingleTriggerEntry<K extends record.Key> extends record.KeyedNamed<K> {
  icon?: Icon.ReactElement;
}

/** Props for {@link SingleTrigger}. */
export interface SingleTriggerProps extends Omit<Dialog.TriggerProps, "variant"> {
  /**
   * The trigger's chassis. A shadow trigger reads as its value: no box at rest, a
   * border on hover, and a box while open. Its caret shows only on hover or while open.
   */
  variant?: Input.Variant;
  /** Haul item type this trigger accepts as a drop. Empty accepts nothing. */
  haulType?: string;
  placeholder?: string;
  icon?: Icon.ReactElement;
  /** Whether to render the icon alone, with no name and no caret. */
  iconOnly?: boolean;
  /**
   * Whether hovering the trigger shows a button that clears the selection, and Delete or
   * Backspace on the focused trigger clears it. The button covers the icon, or the caret
   * when there is no icon. Single sets it from `allowNone`.
   */
  clearable?: boolean;
  /** Chooses an icon from the selected entry, overriding `icon`. */
  renderIcon?: (entry: unknown) => Icon.ReactElement | undefined;
  /**
   * Whether the trigger drops its name and caret when too narrow to read the name. A
   * tooltip then shows the name. The caller must size the select, because the
   * trigger's width no longer depends on the name.
   */
  collapsible?: boolean;
}

const LABEL_SELECTOR = `:scope > .${CSS.BE("select", "label")}`;

const nameReadable = (anchor: HTMLElement): boolean => {
  const label = anchor.querySelector<HTMLElement>(LABEL_SELECTOR);
  if (label == null) return false;
  return (
    getComputedStyle(label).display !== "none" && label.scrollWidth <= label.clientWidth
  );
};

/** The button of a {@link Single} selection, showing the selected entry's name. */
export const SingleTrigger = <K extends record.Key>({
  haulType = "",
  placeholder,
  icon: baseIcon,
  disabled,
  iconOnly = false,
  hideCaret = false,
  variant: triggerVariant = "outlined",
  renderIcon,
  collapsible = false,
  clearable = false,
  preview,
  className,
  status,
  tooltip,
  hideTooltip,
  id: idProp,
  onKeyDown,
  ...rest
}: SingleTriggerProps): ReactElement => {
  const allSelected = useSelected<K>();
  const { setSelected, clear } = useContext<K>();
  const [selected] = allSelected;
  const fixed = useIsFixed(selected);
  const fixedText = useText(selected);
  const item = List.useItem<K, SingleTriggerEntry<K>>(selected);
  const { name, icon } = item ?? {};
  const resolvedIcon = renderIcon?.(item) ?? icon ?? baseIcon;
  const canDrop = (hauled: Haul.DraggingState) =>
    staticCanDrop(hauled, haulType, allSelected, disabled);
  const { startDrag, onDragEnd, ...dropProps } = Haul.useDragAndDrop({
    type: haulType,
    canDrop,
    onDrop: ({ items }) => {
      const dropped = Haul.filterByType(haulType, items);
      if (dropped.length !== 0) setSelected([dropped[0].key as K]);
      return dropped;
    },
  });
  // A drag copies the selection: a single select must keep its value. Only a key that
  // resolves to an entry drags, so an empty sentinel such as 0 never does.
  const draggable =
    haulType !== "" && item != null && !fixed && disabled !== true && preview !== true;
  const handleDragStart = () => {
    if (item != null) startDrag([{ key: item.key, type: haulType }]);
  };
  const dragging = Haul.useDraggingState();
  const baseId = useId();
  const id = idProp ?? baseId;
  const labelId = `${baseId}-label`;
  // A self-reference reads the trigger's own aria-label, so the name becomes the
  // aria-label followed by the selection.
  const shown = fixed ? fixedText : (name ?? (preview === true ? "None" : placeholder));
  const { visible, variant } = Dialog.useContext();
  // A key with no entry, such as a deleted resource, can still be cleared. A zero key
  // means none.
  const canClear =
    clearable && primitive.isNonZero(selected) && disabled !== true && preview !== true;
  const clearOnIcon = canClear && resolvedIcon != null;
  const clearOnCaret =
    canClear && !clearOnIcon && variant !== "modal" && !hideCaret && !iconOnly;
  const clearShown = clearOnIcon || clearOnCaret;
  // The clear button is not a tab stop, so the focused trigger takes the key instead.
  const handleKeyDown = (e: KeyboardEvent<HTMLElement>): void => {
    onKeyDown?.(e);
    if (!canClear || (e.key !== "Delete" && e.key !== "Backspace")) return;
    e.preventDefault();
    clear();
  };
  const withClear = (glyph: ReactElement): ReactElement => (
    <span className={CSS.BE("select-single-trigger", "icon")}>
      {glyph}
      <Button.Close
        el="div"
        role="button"
        aria-label={`Clear ${name ?? fixedText ?? "selection"}`}
        className={CSS.BE("select-single-trigger", "clear")}
        onClick={clear}
      />
    </span>
  );
  const itemStatus = Input.useItemStatus({
    status,
    tooltip: tooltip ?? (collapsible ? shown : undefined),
    hideTooltip: hideTooltip ?? (collapsible ? nameReadable : undefined),
    "aria-invalid": rest["aria-invalid"],
    "aria-describedby": rest["aria-describedby"],
  });
  const labelledBySelection =
    rest["aria-label"] != null &&
    !iconOnly &&
    selected != null &&
    (name != null || fixed);
  return (
    <Dialog.Trigger
      id={id}
      variant={triggerVariant === "shadow" ? "outlined" : triggerVariant}
      gap="small"
      className={CSS.cls(
        CSS.dropRegion(canDrop(dragging)),
        triggerVariant === "shadow" && CSS.M("shadow"),
        name == null && !fixed ? CSS.BM("select-single-trigger", "empty") : null,
        collapsible && CSS.BM("select-single-trigger", "collapsible"),
        clearShown && CSS.M("reveals"),
        clearShown && CSS.BM("select-single-trigger", "clearable"),
        className,
      )}
      disabled={disabled}
      draggable={draggable}
      onDragStart={draggable ? handleDragStart : undefined}
      onDragEnd={onDragEnd}
      {...dropProps}
      {...rest}
      onKeyDown={handleKeyDown}
      aria-labelledby={
        labelledBySelection ? `${id} ${labelId}` : rest["aria-labelledby"]
      }
      status={itemStatus.status}
      aria-invalid={itemStatus["aria-invalid"]}
      aria-describedby={itemStatus["aria-describedby"]}
      tooltip={itemStatus.tooltip}
      hideTooltip={itemStatus.hideTooltip}
      preview={preview}
      hideCaret={hideCaret || iconOnly || clearOnCaret}
    >
      {clearOnIcon ? withClear(resolvedIcon) : resolvedIcon}
      {!iconOnly && (
        <Label itemKey={selected} id={labelId}>
          {shown}
        </Label>
      )}
      {clearOnCaret &&
        withClear(
          <Caret.Animated enabled={visible} enabledLoc="bottom" disabledLoc="left" />,
        )}
    </Dialog.Trigger>
  );
};
