// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/select/MultipleTrigger.css";

import { array, color, primitive, type record, unique } from "@synnaxlabs/x";
import type {
  DragEvent,
  KeyboardEvent,
  MouseEvent,
  ReactElement,
  ReactNode,
} from "react";

import { Button } from "@/button";
import { Caret } from "@/caret";
import { type RenderProp, renderProp } from "@/component/renderProp";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Haul } from "@/haul";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { List } from "@/list";
import { useContext, useReselectNoop, useSelected } from "@/select/Context";
import { Label } from "@/select/Label";
import { plural } from "@/select/placeholder";
import { useIsFixed } from "@/select/registry";
import { preventEnterWhileOpen } from "@/select/use";
import { Tag } from "@/tag";
import { Text } from "@/text";

export interface MultipleEntry<K extends record.Key> extends record.KeyedNamed<K> {
  icon?: Icon.ReactElement;
  color?: color.Crude;
}

export interface MultipleTagProps<K extends record.Key> extends Omit<
  Tag.TagProps,
  "onDragStart"
> {
  itemKey: K;
  /** Removes only this tag's key from the selection, ignoring held modifier keys. */
  onRemove: () => void;
  onDragStart: (key: K, event: DragEvent<HTMLElement>) => void;
  renderIcon?: (entry: unknown) => Icon.ReactElement | undefined;
}

const MultipleTag = <K extends record.Key, E extends MultipleEntry<K>>({
  itemKey,
  icon,
  onDragStart,
  onClick,
  onRemove,
  renderIcon,
}: MultipleTagProps<K>): ReactElement | null => {
  const fixed = useIsFixed(itemKey);
  const item = List.useItem<K, E>(itemKey);
  const label = primitive.isNonZero(item?.name) ? item.name : itemKey.toString();
  const resolvedIcon = fixed ? undefined : (renderIcon?.(item) ?? item?.icon ?? icon);
  return (
    <Tag.Tag
      onClose={onRemove}
      onClick={onClick}
      onDragStart={(event) => onDragStart(itemKey, event)}
      draggable
      size="small"
      status={item == null && !fixed ? "error" : undefined}
      icon={resolvedIcon}
      color={item?.color}
    >
      {fixed ? <Label itemKey={itemKey} /> : label}
    </Tag.Tag>
  );
};

const multipleTag = renderProp(MultipleTag);

/** Props for {@link MultipleTrigger}. */
export interface MultipleTriggerProps<
  K extends record.Key,
  E extends record.Keyed<K> | undefined = MultipleEntry<K> | undefined,
> extends Pick<
  Button.ButtonProps,
  "variant" | "disabled" | "preview" | "className" | "size"
> {
  /** Haul item type this trigger accepts as a drop. Empty accepts nothing. */
  haulType?: string;
  /** Builds the haul item for an entry dragged out of the trigger. */
  createHaulItem?: (entry: NonNullable<E>) => Haul.Item;
  /** Renders the drag preview of an entry. It defaults to the entry's icon and name. */
  renderHaulPreview?: (entry: unknown) => ReactNode;
  placeholder?: ReactNode;
  icon?: Icon.ReactElement;
  /** Names the trigger. The tags alone never say what they are. */
  "aria-label"?: string;
  /** Singular name of what the tags are, such as "label", for a count of them. */
  resourceName?: string;
  /** Whether to show only a count instead of one tag per entry. */
  hideTags?: boolean;
  children?: RenderProp<MultipleTagProps<K>>;
  renderIcon?: (entry: unknown) => Icon.ReactElement | undefined;
}

/**
 * @returns whether a drag carries at least one entry of the type not already selected.
 */
export const staticCanDrop = <K extends record.Key>(
  { items: entities }: Haul.DraggingState,
  haulType: string,
  value: K[] | readonly K[],
  disabled?: boolean,
): boolean => {
  if (haulType === "" || disabled === true) return false;
  const f = Haul.filterByType(haulType, entities);
  return f.length > 0 && !f.every((h) => value.includes(h.key as K));
};

const PLACEHOLDER_CLASS = CSS.B("select-multiple-trigger-placeholder");

/**
 * The button of a {@link Multiple} selection, showing one removable tag per entry. A
 * shadow or text trigger reads as its tags: each tag, the count, and the placeholder of
 * an empty trigger opens the dialog beside itself. The room around them takes no
 * clicks. A shadow trigger's tags have no remove buttons, and in a list item, its
 * placeholder shows on hover only.
 */
export const MultipleTrigger = <
  K extends record.Key,
  E extends record.Keyed<K> | undefined = MultipleEntry<K> | undefined,
>({
  haulType = "",
  createHaulItem,
  renderHaulPreview,
  disabled,
  placeholder = "Select",
  variant = "outlined",
  preview,
  icon,
  "aria-label": ariaLabel,
  resourceName,
  hideTags = false,
  children = multipleTag as unknown as RenderProp<MultipleTagProps<K>>,
  renderIcon,
  className,
  size,
}: MultipleTriggerProps<K, E>): ReactElement | null => {
  const value = useSelected<K>();
  const { getState, setSelected } = useContext<K>();
  // In a multiple frame, a reselect no-op means the selection cannot be empty.
  const reselectNoop = useReselectNoop();
  const { getItem } = List.useUtilContext<K, E>();
  const { toggle, toggleAt, visible, triggerRef } = Dialog.useContext();
  const canDrop = (hauled: Haul.DraggingState) =>
    staticCanDrop(hauled, haulType, array.toArray(value), disabled);
  const { startDrag, ...dropProps } = Haul.useDragAndDrop({
    type: haulType,
    canDrop,
    onDrop: ({ items }) => {
      const dropped = Haul.filterByType(haulType, items);
      setSelected(unique.unique([...value, ...(dropped.map((c) => c.key) as K[])]));
      return dropped;
    },
  });

  const removeKey = (key: K): void => {
    const next = array.toArray(getState().value).filter((k) => k !== key);
    if (next.length === 0 && reselectNoop) return;
    setSelected(next);
  };

  const handleSuccessfulDrop = ({ dropped }: Haul.OnSuccessfulDropProps) => {
    const res = value.filter((key) => !dropped.some((h) => h.key === key));
    setSelected(res);
  };

  const onTagDragStart = (key: K, event: DragEvent<HTMLElement>) => {
    const entry = getItem?.(key);
    let item: Haul.Item = { key, type: haulType };
    if (createHaulItem != null) {
      if (entry == null) return;
      item = createHaulItem(entry);
    }
    const named = entry as MultipleEntry<K> | undefined;
    const name = primitive.isNonZero(named?.name) ? named.name : key.toString();
    startDrag({
      event,
      items: [item],
      preview: renderHaulPreview?.(entry) ?? (
        <Haul.Preview
          icon={renderIcon?.(entry) ?? named?.icon ?? icon}
          swatch={color.cssString(named?.color)}
          name={name}
        />
      ),
      onSuccessfulDrop: handleSuccessfulDrop,
    });
  };
  const dragging = Haul.useDraggingState();
  const showAddButton = variant === "text" && value.length !== 0 && preview !== true;
  const itemStatus = Input.useItemStatus({});
  const handleKeyDown = (e: KeyboardEvent<HTMLElement>): void =>
    preventEnterWhileOpen(e, visible);
  // A shadow or text trigger reads as its tags, and each part opens it beside itself.
  const openAt =
    preview !== true && (variant === "shadow" || variant === "text")
      ? (e: MouseEvent<HTMLElement>): void => toggleAt(e.currentTarget)
      : undefined;

  if (hideTags) {
    if (preview === true) return null;
    return (
      <Dialog.Trigger
        variant={variant}
        aria-label={ariaLabel}
        className={className}
        onKeyDown={handleKeyDown}
        {...dropProps}
        {...itemStatus}
      >
        {icon}
        {placeholder}
      </Dialog.Trigger>
    );
  }

  return (
    <Tag.Tags
      // A ref that changes after mount never runs, so the box mounts again when the
      // add button comes or goes.
      key={showAddButton ? "add" : "box"}
      ref={showAddButton ? undefined : triggerRef}
      full="x"
      // The chassis is a div, so it needs a role and a tab stop to act as a button.
      role="button"
      tabIndex={showAddButton || preview === true ? undefined : 0}
      aria-label={ariaLabel}
      onClick={toggle}
      onCountClick={openAt}
      noun={
        resourceName == null
          ? undefined
          : { one: resourceName, other: plural(resourceName) }
      }
      onKeyDown={handleKeyDown}
      {...dropProps}
      className={CSS.cls(
        CSS.dropRegion(canDrop(dragging)),
        CSS.BE("dialog", "trigger"),
        CSS.BM("variant", variant),
        className,
      )}
      variant={variant}
      preview={preview}
      grow
      {...itemStatus}
    >
      {value.length === 0 &&
        (preview !== true && (variant === "shadow" || variant === "text") ? (
          <Button.Button
            el="div"
            variant="text"
            size={size}
            onClick={openAt}
            reveal={variant === "shadow"}
            className={PLACEHOLDER_CLASS}
          >
            {icon}
            {placeholder}
          </Button.Button>
        ) : (
          <Text.Text
            className={CSS.cls(
              PLACEHOLDER_CLASS,
              variant === "shadow" && CSS.M("reveal"),
            )}
          >
            {preview === true ? (
              "None"
            ) : (
              <>
                {icon}
                {placeholder}
              </>
            )}
          </Text.Text>
        ))}
      {value.map((v) =>
        children({
          key: v,
          itemKey: v,
          onRemove: () => removeKey(v),
          onDragStart: onTagDragStart,
          onClick: openAt,
          icon,
          renderIcon,
        }),
      )}
      {variant === "outlined" && preview !== true && (
        <Caret.Animated
          className={CSS.level("p")}
          enabled={visible}
          enabledLoc="bottom"
          disabledLoc="left"
          color={9}
        />
      )}
      {showAddButton && (
        <Button.Button ref={triggerRef} variant="text" size={size} onClick={openAt}>
          <Icon.Add color={9} />
        </Button.Button>
      )}
    </Tag.Tags>
  );
};
