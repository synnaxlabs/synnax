// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import clsx from "clsx";
import { AiFillCaretDown, AiFillCaretRight } from "react-icons/ai";

import { ButtonIconOnlyProps } from "@/core/Button";
import { Header } from "@/core/Header";
import { Resize, ResizeMultipleProps } from "@/core/Resize";
import { Direction } from "@/spatial";
import { expandedCls } from "@/util/css";

import "./Accordion.css";

export interface AccordionEntry {
  key: string;
  name: string;
  content: ReactElement;
  actions?: Array<ButtonIconOnlyProps | ReactElement>;
}

export interface AccordionProps
  extends Omit<
    ResizeMultipleProps,
    "sizeDistribution" | "parentSize" | "onDragHandle"
  > {
  entries: AccordionEntry[];
  direction?: Direction;
}

export const Accordion = ({
  direction = "y",
  entries,
}: AccordionProps): JSX.Element => {
  const {
    setSize,
    props: { sizeDistribution: sizes, parentSize, ...resizeProps },
  } = Resize.useMultiple({
    direction,
    count: entries.length,
    minSize: 28,
  });

  const onExpand = (index: number): void => {
    if (sizes[index] < 40 / parentSize) setSize(index, 200);
    else setSize(index, 28);
  };

  return (
    <Resize.Multiple
      empty
      className="pluto-accordion"
      sizeDistribution={sizes}
      parentSize={parentSize}
      {...resizeProps}
    >
      {entries.map((entry, i) => (
        <AccordionEntryC
          {...entry}
          key={entry.key}
          index={i}
          direction={direction}
          onExpand={onExpand}
          expanded={sizes[i] * parentSize > 32}
        />
      ))}
    </Resize.Multiple>
  );
};

interface AccordionEntryCProps extends Omit<AccordionEntry, "key"> {
  index: number;
  expanded: boolean;
  onExpand: (i: number) => void;
  direction: Direction;
}

const AccordionEntryC = ({
  index,
  name,
  content,
  direction,
  actions,
  expanded,
  onExpand,
}: AccordionEntryCProps): JSX.Element => (
  <>
    <Header
      level="p"
      className={clsx(
        "pluto-accordion__header",
        `pluto-accordion__header--${expandedCls(expanded)}`
      )}
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
);
