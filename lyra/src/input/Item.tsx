// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/Item.css";

import { direction, primitive } from "@synnaxlabs/x";
import {
  type AriaAttributes,
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
  useId,
} from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { HelpText } from "@/input/HelpText";
import { Label } from "@/input/Label";
import { ItemContext, type ItemContextValue, useItemContext } from "@/input/scope";
import { type Status } from "@/status";
import { Text } from "@/text";
import { type Tooltip } from "@/tooltip";

/**
 * Resolves a control's aria-labelledby: an explicit value wins, then the enclosing
 * {@link Item}'s label id, unless the control carries its own aria-label.
 */
export const useLabelledBy = (props: {
  "aria-label"?: string;
  "aria-labelledby"?: string;
}): string | undefined => {
  const labelID = useItemContext()?.labelID;
  return (
    props["aria-labelledby"] ?? (props["aria-label"] == null ? labelID : undefined)
  );
};

/**
 * @returns whether an enclosing {@link Item} shows a visible label for the control.
 * A control that carries its own name must say what it is when it has no value.
 */
export const useLabelled = (): boolean => useItemContext()?.labelID != null;

/** The props through which a control shows the status of its {@link Item}. */
export interface ItemStatusProps extends Pick<
  Tooltip.ExtensionProps,
  "tooltip" | "hideTooltip"
> {
  status?: Status.Variant;
  "aria-invalid"?: AriaAttributes["aria-invalid"];
  "aria-describedby"?: string;
}

const isMarked = (variant?: Status.Variant): boolean =>
  variant === "error" || variant === "warning";

/**
 * Resolves the status props of a control in an {@link Item}. The control's own props
 * win, except its tooltip: when the item has no help text, the item's error or warning
 * shows in the tooltip, and the control takes the item's status as its own. That
 * tooltip stays mounted while it is empty, so a status that comes or goes never
 * remounts the control.
 */
export const useItemStatus = (props: ItemStatusProps): ItemStatusProps => {
  const item = useItemContext();
  if (item == null) return props;
  const { status, message, messageID, helpTextHidden } = item;
  const resolved: ItemStatusProps = {
    ...props,
    "aria-invalid": props["aria-invalid"] ?? (status === "error" ? true : undefined),
  };
  if (!helpTextHidden) {
    resolved["aria-describedby"] =
      props["aria-describedby"] ??
      (primitive.isNonZero(message) ? messageID : undefined);
    return resolved;
  }
  const marked = isMarked(status);
  resolved.status = props.status ?? (marked ? status : undefined);
  const tip = marked && primitive.isNonZero(message) ? message : undefined;
  if (tip == null && props.tooltip != null) return resolved;
  // A text element tip, unlike a string, never names an icon-only control.
  resolved.tooltip = (
    <Text.Text level="small" color={11}>
      {tip}
    </Text.Text>
  );
  resolved.hideTooltip = tip == null;
  return resolved;
};

/**
 * Hides the enclosing {@link Item} from controls that are not its own, such as the
 * fields in a control's popover.
 */
export const ItemBoundary = ({ children }: PropsWithChildren): ReactElement => (
  <ItemContext value={null}>{children}</ItemContext>
);

export interface ItemProps extends Flex.BoxProps {
  label?: string;
  required?: boolean;
  showLabel?: boolean;
  helpText?: string;
  padHelpText?: boolean;
  status?: Status.Variant;
  showHelpText?: boolean;
}

const maybeDefaultAlignment = (
  align?: Flex.Alignment,
  dir: direction.Crude = "y",
): Flex.Alignment => {
  if (align != null) return align;
  return direction.construct(dir) === "y" ? "stretch" : "center";
};

/**
 * A control with its label and help text, which shows the status message. With no
 * help text, the control shows the status itself. The layout follows the props, never
 * the message, so a message that comes or goes never remounts the control.
 */
export const Item = ({
  label,
  showLabel = true,
  helpText,
  direction,
  x,
  y,
  className,
  children,
  required,
  align,
  gap: size = "small",
  padHelpText = false,
  status,
  showHelpText = true,
  ...rest
}: ItemProps): ReactNode => {
  const dir = Flex.parseDirection(direction, x, y, false);
  const labelID = useId();
  const messageID = `${labelID}-help`;
  const labelShown = showLabel && primitive.isNonZero(label);
  const value: ItemContextValue = {
    labelID: labelShown ? labelID : undefined,
    status,
    message: helpText,
    messageID,
    helpTextHidden: !showHelpText,
  };
  if (!labelShown && !showHelpText)
    return <ItemContext value={value}>{children}</ItemContext>;
  const help = showHelpText && (padHelpText || primitive.isNonZero(helpText)) && (
    <HelpText id={messageID} variant={status}>
      {helpText}
    </HelpText>
  );
  return (
    <ItemContext value={value}>
      <Flex.Box
        className={CSS.cls(CSS.BE("input", "item"), className)}
        direction={dir}
        gap={size}
        align={maybeDefaultAlignment(align, dir)}
        {...rest}
      >
        {labelShown && (
          <Label id={labelID} required={required}>
            {label}
          </Label>
        )}
        {dir === "x" ? (
          <Flex.Box y gap="small">
            {children}
            {help}
          </Flex.Box>
        ) : (
          <Flex.Box y gap={1 / 3} align="stretch">
            {children}
            {help}
          </Flex.Box>
        )}
      </Flex.Box>
    </ItemContext>
  );
};
