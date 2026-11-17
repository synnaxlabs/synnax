// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/tag/Tags.css";

import {
  createContext,
  type MouseEvent,
  type PropsWithChildren,
  type ReactElement,
  type RefObject,
  useContext,
  useLayoutEffect,
  useRef,
  useState,
} from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { useCombinedRefs } from "@/hooks";
import { Icon } from "@/icon";
import { Steps } from "@/tag/step";
import { Text } from "@/text";
import { Tooltip } from "@/tooltip";

export interface TagsProps extends Button.ButtonProps<"div"> {
  /**
   * Called on a click on the count. With it, only the group's tags, count, and buttons
   * take the pointer, and each tag and the count lights on hover and press.
   */
  onCountClick?: (e: MouseEvent<HTMLElement>) => void;
  /**
   * What the tags are, in the singular and the plural. With it, a count with no tags
   * beside it names the total, such as "3 labels". Without it, the count reads "+3".
   */
  noun?: { one: string; other: string };
}

interface Hidden {
  name: string;
  color?: string;
}

// A tag's name, and the color of its dot if it has one.
const describe = (tag: Element): Hidden => {
  const icon = tag.querySelector(`:scope > .${CSS.BM("icon", "dot")}`);
  return {
    name: tag.textContent ?? "",
    color: icon == null ? undefined : getComputedStyle(icon).color,
  };
};

interface GroupProps {
  groupRef: RefObject<HTMLElement | null>;
}

// The tags a group hides, read when the list opens.
const HiddenList = ({ groupRef }: GroupProps): ReactElement => {
  const [hidden, setHidden] = useState<Hidden[]>([]);
  useLayoutEffect(() => {
    const el = groupRef.current;
    if (el == null) return;
    const tags = Array.from(el.children).filter(
      (c) => c.classList.contains(CSS.B("tag")) && c.hasAttribute("data-hidden"),
    );
    setHidden(tags.map(describe));
  }, [groupRef]);
  return (
    <div className={CSS.BE("tags", "hidden")}>
      {hidden.map(({ name, color }) => (
        <Text.Text key={name} level="small" gap="small">
          {color != null && <Icon.Circle color={color} />}
          {name}
        </Text.Text>
      ))}
    </div>
  );
};

interface MoreProps extends GroupProps, Pick<TagsProps, "onCountClick" | "noun"> {}

/** The count of a group. Its styles show the number its step hides. */
const More = ({ groupRef, onCountClick, noun }: MoreProps): ReactElement => (
  <Tooltip.Dialog>
    <HiddenList groupRef={groupRef} />
    <Text.Text
      level="small"
      className={CSS.BE("tags", "more")}
      data-one={noun?.one}
      data-other={noun?.other}
      // A count that acts keeps its click from the group, as a tag does.
      onClick={
        onCountClick == null
          ? undefined
          : (e: MouseEvent<HTMLElement>) => {
              e.stopPropagation();
              onCountClick(e);
            }
      }
    />
  </Tooltip.Dialog>
);

const StepContext = createContext<Steps | null>(null);
StepContext.displayName = "Tag.StepContext";

/**
 * Fits the groups of tags under it together, so they all show the same step: the same
 * number of tags, then a count.
 */
export const StepProvider = ({ children }: PropsWithChildren): ReactElement => {
  const [steps] = useState(() => new Steps());
  const ref = useRef<HTMLDivElement | null>(null);
  useLayoutEffect(() => {
    if (ref.current == null) return;
    steps.attach(ref.current);
    return () => steps.detach();
  }, [steps]);
  return (
    <div ref={ref} className={CSS.BE("tags", "steps")}>
      <StepContext value={steps}>{children}</StepContext>
    </div>
  );
};

/**
 * A row of tags that never overflows its container. Tags that do not fit stack, then
 * fold into a count that lists them on hover. Under a {@link StepProvider}, it shows
 * the step the provider sets.
 */
export const Tags = ({
  ref,
  children,
  variant = "outlined",
  size = "medium",
  className,
  onCountClick,
  noun,
  ...rest
}: TagsProps) => {
  const shared = useContext(StepContext);
  // A group outside a provider fits alone.
  const [own] = useState(() => (shared == null ? new Steps() : null));
  const elRef = useRef<HTMLDivElement | null>(null);
  const combinedRef = useCombinedRefs(ref, elRef);
  useLayoutEffect(() => {
    const el = elRef.current;
    const steps = shared ?? own;
    if (el == null || steps == null) return;
    own?.attach(el);
    steps.add(el);
    return () => {
      steps.remove(el);
      own?.detach();
    };
  }, [shared, own]);
  return (
    <Button.Button
      ref={combinedRef}
      className={CSS.cls(
        CSS.B("tags"),
        shared != null && CSS.BM("tags", "stepped"),
        onCountClick != null && CSS.BM("tags", "parts"),
        className,
      )}
      el="div"
      variant={variant}
      size={size}
      x
      align="center"
      gap="small"
      rounded="small"
      {...rest}
    >
      {children}
      <More groupRef={elRef} onCountClick={onCountClick} noun={noun} />
    </Button.Button>
  );
};
