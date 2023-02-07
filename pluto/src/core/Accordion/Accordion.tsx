// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, RefObject } from "react";

import clsx from "clsx";
import { AiFillCaretDown, AiFillCaretRight } from "react-icons/ai";

import { ButtonIconOnlyProps } from "@/core/Button";
import { Header } from "@/core/Header";
import { Resize, ResizeMultipleProps } from "@/core/Resize";
import { Box, Direction } from "@/spatial";
import { expandedCls } from "@/util/css";

import "./Accordion.css";

/** The props for a single entry in the {@link Accordion} component. */
export interface AccordionEntry {
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
  actions?: Array<ButtonIconOnlyProps | ReactElement>;
}

/** The props for the {@link Accordion} component. */
export interface AccordionProps
  extends Omit<
    ResizeMultipleProps,
    "sizeDistribution" | "parentSize" | "onDragHandle" | "direction"
  > {
  data: AccordionEntry[];
}

const DIRECTION: Direction = "y";
const MIN_SIZE = 28;
const COLLAPSED_THRESHOLD = 32;
const EXPAND_THRESHOLD = 40;
const DEFAULT_EXPAND_SIZE = 0.5;

/**
 * A resizable accordion component, whose entries can be expanded and collapsed. This
 * component is intentionally constrained in its interface in order to provide stylistic
 * consistency and simplicity. If you need more control, look at building a custom
 * accordion component using {@link Resize.Multiple}.
 *
 * @param props - All unused props are passed to the underyling {@link Resize.Multiple}
 * component.
 * @param props.entries - The entries to display in the accordion. See the
 * {@link AccordionEntry} interface for more details.
 */
export const Accordion = ({ data, ...props }: AccordionProps): JSX.Element => {
  const {
    setSize,
    props: { sizeDistribution: sizes, ref, ...resizeProps },
  } = Resize.useMultiple({
    direction: DIRECTION,
    count: data.length,
    minSize: MIN_SIZE,
  });

  const onExpand = (index: number): void => {
    if(ref.current == null) return;
    const parentSize = new Box(ref.current).dim(DIRECTION);
    if (sizes[index] < EXPAND_THRESHOLD / parentSize)
      setSize(index, data[index].initialSize ?? DEFAULT_EXPAND_SIZE);
    else setSize(index, MIN_SIZE + 1);
  };

  return (
    <Resize.Multiple
      empty
      className="pluto-accordion"
      sizeDistribution={sizes}
      ref={ref}
      {...props}
      {...resizeProps}
    >
      {data.map((entry, i) => (
        <AccordionEntryC
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

interface AccordionEntryCProps extends Omit<AccordionEntry, "key"> {
  index: number;
  size: number;
  parent: RefObject<HTMLDivElement>;
  onExpand: (i: number) => void;
  direction: Direction;
}

const AccordionEntryC = ({
  index,
  name,
  content,
  actions,
  size,
  parent,
  onExpand,
}: AccordionEntryCProps): JSX.Element => {
  let expanded = true
  if(parent.current != null) {
      const parentSize = new Box(parent.current).dim(DIRECTION);
     expanded = size * parentSize > COLLAPSED_THRESHOLD;
  } 
    return (
    <>
      <Header
        level="p"
        className={clsx("pluto-accordion__header", expandedCls(expanded))}
        empty
      >
        <Header.ButtonTitle
          startIcon={expanded ? <AiFillCaretDown /> : <AiFillCaretRight />}
          onClick={() => onExpand(index)}
        >
          {name}
        </Header.ButtonTitle>
        {actions != null && <Header.Actions>{actions}</Header.Actions>}
      </Header>
      {content}
    </>
  )
};
