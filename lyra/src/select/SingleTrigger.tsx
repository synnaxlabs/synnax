// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/select/SingleTrigger.css";

import { type record } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useId } from "react";

import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Haul } from "@/haul";
import { type Icon } from "@/icon";
import { List } from "@/list";
import { useContext, useSelected } from "@/select/Context";
import { Label } from "@/select/Label";
import { staticCanDrop } from "@/select/MultipleTrigger";
import { useIsFixed, useText } from "@/select/registry";

export interface SingleTriggerEntry<K extends record.Key> extends record.KeyedNamed<K> {
  icon?: Icon.ReactElement;
}

/** Props for {@link SingleTrigger}. */
export interface SingleTriggerProps extends Dialog.TriggerProps {
  /** Haul item type this trigger accepts as a drop. Empty accepts nothing. */
  haulType?: string;
  placeholder?: string;
  icon?: Icon.ReactElement;
  /** Whether to render the icon alone, with no name and no caret. */
  iconOnly?: boolean;
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
  renderIcon,
  collapsible = false,
  preview,
  className,
  tooltip,
  hideTooltip,
  id: idProp,
  ...rest
}: SingleTriggerProps): ReactElement => {
  const allSelected = useSelected<K>();
  const { setSelected } = useContext<K>();
  const [selected] = allSelected;
  const fixed = useIsFixed(selected);
  const fixedText = useText(selected);
  const item = List.useItem<K, SingleTriggerEntry<K>>(selected);
  const { name, icon } = item ?? {};
  const resolvedIcon = renderIcon?.(item) ?? icon ?? baseIcon;
  const canDrop = useCallback(
    (hauled: Haul.DraggingState) =>
      staticCanDrop(hauled, haulType, allSelected, disabled),
    [haulType, allSelected, disabled],
  );
  const dropProps = Haul.useDrop({
    type: haulType,
    canDrop,
    onDrop: Haul.useFilterByTypeCallback(
      haulType,
      ({ items }) => {
        if (items.length !== 0) setSelected([items[0].key as K]);
        return items;
      },
      [setSelected],
    ),
  });
  const dragging = Haul.useDraggingState();
  const baseId = useId();
  const id = idProp ?? baseId;
  const labelId = `${baseId}-label`;
  // A self-reference reads the trigger's own aria-label, so the name becomes the
  // aria-label followed by the selection.
  const shown = fixed ? fixedText : (name ?? (preview === true ? "None" : placeholder));
  const labelledBySelection =
    rest["aria-label"] != null &&
    !iconOnly &&
    selected != null &&
    (name != null || fixed);
  return (
    <Dialog.Trigger
      id={id}
      variant="outlined"
      gap="small"
      className={CSS.cls(
        CSS.dropRegion(canDrop(dragging)),
        name == null && !fixed ? CSS.BM("select-single-trigger", "empty") : null,
        collapsible && CSS.BM("select-single-trigger", "collapsible"),
        className,
      )}
      disabled={disabled}
      {...dropProps}
      {...rest}
      aria-labelledby={
        labelledBySelection ? `${id} ${labelId}` : rest["aria-labelledby"]
      }
      tooltip={tooltip ?? (collapsible ? shown : undefined)}
      hideTooltip={hideTooltip ?? (collapsible ? nameReadable : undefined)}
      preview={preview}
      hideCaret={hideCaret || iconOnly}
    >
      {resolvedIcon}
      {!iconOnly && (
        <Label itemKey={selected} id={labelId}>
          {shown}
        </Label>
      )}
    </Dialog.Trigger>
  );
};
