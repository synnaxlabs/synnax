// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/accordion/Accordion.css";

import { Icon } from "@synnaxlabs/media";
import { box, type direction } from "@synnaxlabs/x/spatial";
import { type ReactElement, type RefObject } from "react";

import { type Button } from "@/button";
import { CSS } from "@/css";
import { Header } from "@/header";
import { Resize } from "@/resize";

/** The props for a single entry in the {@link Accordion} component. */
export interface Entry {
  /** A unique key for the entry. */
  key: string;
  /** The name for the entries header. */
  name: string;
  /** The content for the entry. */
  content: ReactElement;
  /**
   * The initial size for the entry. It's recommended to set this to a decimal value
   * representing the percentage of the overall parent size.
   */
  initialSize?: number;
  /**
   * A list of actions to display in the entry's header. See the {@link Header.Actions}
   * component for more details.
   */
  actions?: Array<Button.IconProps | ReactElement>;
}

/** The props for the {@link Accordion} component. */
export interface AccordionProps
  extends Omit<
    Resize.MultipleProps,
    "sizeDistribution" | "parentSize" | "onDragHandle" | "direction"
  > {
  data: Entry[];
}

const DIRECTION: direction.Direction = "y";
const MIN_SIZE = 28;
const COLLAPSED_THRESHOLD = 32;
const EXPAND_THRESHOLD = 40;
const DEFAULT_EXPAND_SIZE = 0.5;

/**
 * A resizable accordion component, whose entries can be expanded and collapsed. This
 * component is intentionally opinionated in its interface in order to provide stylistic
 * consistency and simplicity. If you need more control, look at building a custom
 * accordion component using {@link Resize.Multiple}.
 *
 * @param props - All unused props are passed to the underlying {@link Resize.Multiple}
 * component.
 * @param props.entries - The entries to display in the accordion. See the
 * {@link Entry} interface for more details.
 */
export const Accordion = ({ data, ...rest }: AccordionProps): ReactElement => {
  const {
    setSize,
    props: { sizeDistribution: sizes, ref, ...resizeProps },
  } = Resize.useMultiple({
    direction: DIRECTION,
    count: data.length,
    minSize: MIN_SIZE,
  });

  const onExpand = (index: number): void => {
    if (ref.current == null) return;
    const parentSize = box.dim(box.construct(ref.current), DIRECTION);
    if (sizes[index] < EXPAND_THRESHOLD / parentSize)
      setSize(index, data[index].initialSize ?? DEFAULT_EXPAND_SIZE);
    else setSize(index, MIN_SIZE + 1);
  };

  return (
    <Resize.Multiple
      empty
      className={CSS.B("accordion")}
      sizeDistribution={sizes}
      ref={ref}
      {...rest}
      {...resizeProps}
    >
      {data.map((entry, i) => (
        <EntryC
          {...entry}
          key={entry.key}
          index={i}
          direction={DIRECTION}
          onExpand={onExpand}
          parent={ref}
          size={sizes[i]}
        />
      ))}
    </Resize.Multiple>
  );
};

interface EntryCProps extends Omit<Entry, "key"> {
  index: number;
  size: number;
  parent: RefObject<HTMLDivElement | null>;
  onExpand: (i: number) => void;
  direction: direction.Direction;
}

const expandedIcon = <Icon.Caret.Down aria-label="contract" />;
const collapsedIcon = <Icon.Caret.Right aria-label="expand" />;

const EntryC = ({
  index,
  name,
  content,
  actions,
  size,
  parent,
  onExpand,
}: EntryCProps): ReactElement => {
  let expanded = true;
  if (parent.current != null) {
    const parentSize = box.dim(box.construct(parent.current), DIRECTION);
    expanded = size * parentSize > COLLAPSED_THRESHOLD;
  }
  const icon = expanded ? expandedIcon : collapsedIcon;
  return (
    <>
      <Header.Header level="p" className={CSS.expanded(expanded)} empty>
        <Header.ButtonTitle startIcon={icon} onClick={() => onExpand(index)}>
          {name}
        </Header.ButtonTitle>
        {actions != null && <Header.Actions>{actions}</Header.Actions>}
      </Header.Header>
      {content}
    </>
  );
};
