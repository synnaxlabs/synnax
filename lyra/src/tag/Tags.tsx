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
  type PropsWithChildren,
  type ReactElement,
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
import { Batch, Context, type Count, Group, type Hidden, stackHost } from "@/tag/fold";
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

/**
 * Fits the groups of tags under it in one pass per frame. A group with no provider
 * above it fits on its own.
 */
export const FoldProvider = ({ children }: PropsWithChildren): ReactElement => {
  const [batch] = useState(() => new Batch());
  useEffect(() => () => batch.dispose(), [batch]);
  return <Context value={batch}>{children}</Context>;
};

/**
 * A row of tags that never overflows its container. Tags that do not fit stack, then
 * fold into a count that lists them on hover.
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
  const shared = useContext(Context);
  const [own] = useState(() => (shared == null ? new Batch() : null));
  const batch = shared ?? own;
  useEffect(() => () => own?.dispose(), [own]);
  const [fold, setFold] = useState<Fold>({ hidden: [], count: "none" });
  const elRef = useRef<HTMLDivElement | null>(null);
  const groupRef = useRef<Group | null>(null);
  const combinedRef = useCombinedRefs(ref, elRef);
  useLayoutEffect(() => {
    const el = elRef.current;
    if (el == null || batch == null) return;
    const group = new Group(el, size, (hidden, count) => {
      const next = { hidden, count };
      setFold((prev) => (sameFold(prev, next) ? prev : next));
    });
    groupRef.current = group;
    batch.add(group);
    return () => {
      batch.remove(group);
      groupRef.current = null;
    };
  }, [batch, size]);
  // A refit while the item animates would read a width mid-way.
  useEffect(() => {
    const el = elRef.current;
    const host = el == null ? null : stackHost(el);
    if (host == null) return;
    const isShrink = (e: TransitionEvent): boolean =>
      e.target === host && e.propertyName === "flex-shrink";
    const start = (e: TransitionEvent): void => {
      if (isShrink(e) && groupRef.current != null) groupRef.current.animating = true;
    };
    const end = (e: TransitionEvent): void => {
      const group = groupRef.current;
      if (!isShrink(e) || group == null) return;
      group.animating = false;
      batch?.request(group);
    };
    host.addEventListener("transitionrun", start);
    host.addEventListener("transitionend", end);
    host.addEventListener("transitioncancel", end);
    return () => {
      host.removeEventListener("transitionrun", start);
      host.removeEventListener("transitionend", end);
      host.removeEventListener("transitioncancel", end);
    };
  }, [batch]);
  return (
    <Button.Button
      ref={combinedRef}
      className={CSS.cls(CSS.B("tags"), className)}
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
      {fold.count !== "none" && <More hidden={fold.hidden} count={fold.count} />}
    </Button.Button>
  );
};
