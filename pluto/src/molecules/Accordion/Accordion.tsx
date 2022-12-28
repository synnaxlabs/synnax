import { ReactElement } from "react";

import { AiFillCaretDown, AiFillCaretRight } from "react-icons/ai";

import { Space } from "../../atoms/Space";

import { ButtonIconOnlyProps, Header, Resize } from "@/atoms";
import { Direction } from "@/util/spatial";

export interface AccordionEntry {
  key: string;
  title: string;
  content: ReactElement;
  actions?: Array<ButtonIconOnlyProps | ReactElement>;
}

export interface AccordionProps {
  entries: AccordionEntry[];
  direction?: Direction;
}

export const Accordion = ({
  direction = "vertical",
  entries,
}: AccordionProps): JSX.Element => {
  const {
    setSize,
    props: { sizes, ...resizeProps },
  } = Resize.useMultiple({
    direction,
    count: entries.length,
    minSize: 27,
  });

  const onExpand = (index: number): void => {
    if (sizes[index] < 40) setSize(index, undefined, 200);
    else setSize(index, undefined, 28);
  };

  return (
    <Resize.Multiple empty style={{ height: "100%" }} sizes={sizes} {...resizeProps}>
      {entries.map((entry, i) => (
        <AccordionEntryC
          {...entry}
          key={entry.key}
          direction={direction}
          onExpand={onExpand}
          index={i}
          size={sizes[i]}
        />
      ))}
    </Resize.Multiple>
  );
};

interface AccordionEntryCProps extends Omit<AccordionEntry, "key"> {
  index: number;
  size: number;
  onExpand: (i: number) => void;
  direction: Direction;
}

const AccordionEntryC = ({
  index,
  title,
  content,
  direction,
  actions,
  onExpand,
  size,
}: AccordionEntryCProps): JSX.Element => {
  const expanded = size > 28;
  return (
    <Space direction={direction} empty style={{ height: "100%" }}>
      <Header
        level="p"
        style={{ borderBottom: expanded ? "var(--pluto-border)" : "none" }}
      >
        <Header.ButtonTitle
          startIcon={expanded ? <AiFillCaretDown /> : <AiFillCaretRight />}
          onClick={() => onExpand(index)}
          style={{
            borderRadius: "0px",
          }}
        >
          {title}
        </Header.ButtonTitle>
        {actions != null && <Header.Actions>{actions}</Header.Actions>}
      </Header>
      {content}
    </Space>
  );
};
