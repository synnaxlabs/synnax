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
  type PropsWithChildren,
  type ReactElement,
  type RefObject,
  useContext,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { useCombinedRefs } from "@/hooks";
import { Icon } from "@/icon";
import { Batch, type Count, describe, Group, type Hidden } from "@/tag/fold";
import { Steps } from "@/tag/step";
import { Text } from "@/text";
import { Tooltip } from "@/tooltip";

export interface TagsProps extends Button.ButtonProps<"div"> {}

const HiddenList = ({ hidden }: { hidden: Hidden[] }): ReactElement => (
  <div className={CSS.BE("tags", "hidden")}>
    {hidden.map(({ name, color }) => (
      <Text.Text key={name} level="small" gap="small">
        {color != null && <Icon.Circle color={color} />}
        {name}
      </Text.Text>
    ))}
  </div>
);

interface MoreProps {
  hidden: Hidden[];
  count: Count;
}

const More = ({ hidden, count }: MoreProps): ReactElement => {
  const dots = hidden.filter((h) => h.color != null);
  return (
    <Tooltip.Dialog>
      <HiddenList hidden={hidden} />
      <Text.Text level="small" className={CSS.BE("tags", "more")}>
        {dots.length > 0 && (
          <span className={CSS.BE("tags", "dots")}>
            {dots.slice(0, 3).map(({ name, color }) => (
              <Icon.Circle key={name} color={color} />
            ))}
          </span>
        )}
        +{hidden.length}
        {count === "wide" && dots.length === hidden.length ? " labels" : ""}
      </Text.Text>
    </Tooltip.Dialog>
  );
};

interface Fold {
  hidden: Hidden[];
  count: Count;
}

const sameFold = (a: Fold, b: Fold): boolean =>
  a.count === b.count &&
  a.hidden.length === b.hidden.length &&
  a.hidden.every(
    (h, i) => h.name === b.hidden[i].name && h.color === b.hidden[i].color,
  );

// The tags a stepped group hides, read when the list opens.
const SteppedHidden = ({
  groupRef,
}: {
  groupRef: RefObject<HTMLElement | null>;
}): ReactElement => {
  const [hidden, setHidden] = useState<Hidden[]>([]);
  useLayoutEffect(() => {
    const el = groupRef.current;
    if (el == null) return;
    const tags = Array.from(el.children).filter(
      (c) => c.classList.contains(CSS.B("tag")) && c.hasAttribute("data-hidden"),
    );
    setHidden(tags.map((t) => describe(t as HTMLElement)));
  }, [groupRef]);
  return <HiddenList hidden={hidden} />;
};

/** The count of a stepped group. Its styles show the number its step hides. */
const SteppedMore = ({
  groupRef,
}: {
  groupRef: RefObject<HTMLElement | null>;
}): ReactElement => (
  <Tooltip.Dialog>
    <SteppedHidden groupRef={groupRef} />
    <Text.Text level="small" className={CSS.BE("tags", "more")} />
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
  onClick,
  variant = "outlined",
  size = "medium",
  className,
  ...rest
}: TagsProps) => {
  const steps = useContext(StepContext);
  const stepped = steps != null;
  const [batch] = useState(() => (stepped ? null : new Batch()));
  useEffect(() => () => batch?.dispose(), [batch]);
  const [fold, setFold] = useState<Fold>({ hidden: [], count: "none" });
  const elRef = useRef<HTMLDivElement | null>(null);
  const combinedRef = useCombinedRefs(ref, elRef);
  useLayoutEffect(() => {
    const el = elRef.current;
    if (el == null || batch == null) return;
    const group = new Group(el, size, (hidden, count) => {
      const next = { hidden, count };
      setFold((prev) => (sameFold(prev, next) ? prev : next));
    });
    batch.add(group);
    return () => batch.remove(group);
  }, [batch, size]);
  useLayoutEffect(() => {
    const el = elRef.current;
    if (el == null || steps == null) return;
    steps.add(el);
    return () => steps.remove(el);
  }, [steps]);
  return (
    <Button.Button
      ref={combinedRef}
      className={CSS.cls(
        CSS.B("tags"),
        stepped && CSS.BM("tags", "stepped"),
        className,
      )}
      el="div"
      variant={variant}
      size={size}
      x
      onClick={onClick}
      align="center"
      gap="small"
      rounded="small"
      {...rest}
    >
      {children}
      {stepped ? (
        <SteppedMore groupRef={elRef} />
      ) : (
        fold.count !== "none" && <More hidden={fold.hidden} count={fold.count} />
      )}
    </Button.Button>
  );
};
