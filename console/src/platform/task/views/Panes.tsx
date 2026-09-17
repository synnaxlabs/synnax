// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/task/views/Panes.css";

import { Button, Divider, Flex, Header, Icon, useResize } from "@synnaxlabs/pluto";
import { box } from "@synnaxlabs/x";
import {
  type MouseEvent,
  type ReactNode,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";

import { CSS } from "@/platform/css";
import { DetailsHeader } from "@/platform/task/views/DetailsHeader";

export interface PanesProps {
  /** Names the list in its header: "Channels", "Endpoints". */
  listTitle: string;
  /** The list column's items, below the header. */
  list: ReactNode;
  /** The form path of the selected item, or null when nothing is selected. */
  detailsPath: string | null;
  /** Names the selected item in the details header. */
  title?: ReactNode;
  /** The details body, or a placeholder when nothing is selected. */
  children: ReactNode;
}

/** Below this width the list leaves the layout and opens as a drawer instead. */
const NARROW_WIDTH = 560;

/**
 * A list beside the details of its selected item. A toggle in the list's corner
 * hides and shows it. When the form is too narrow for both, the list leaves the
 * layout on its own and the toggle opens it as a drawer over the details; a new
 * selection closes the drawer.
 */
export const Panes = ({
  listTitle,
  list,
  detailsPath,
  title,
  children,
}: PanesProps) => {
  const [narrow, setNarrow] = useState(false);
  const narrowRef = useRef(narrow);
  // A mode switch lands without motion: transitions are off for the commit that
  // switches and back on two frames later, once the new position has painted.
  const [snap, setSnap] = useState(false);
  const ref = useResize<HTMLDivElement>(
    useCallback((b) => {
      const next = box.width(b) < NARROW_WIDTH;
      if (next === narrowRef.current) return;
      narrowRef.current = next;
      setNarrow(next);
      setSnap(true);
    }, []),
    { triggers: ["x"] },
  );
  useEffect(() => {
    if (!snap) return;
    let inner = 0;
    const outer = requestAnimationFrame(() => {
      inner = requestAnimationFrame(() => setSnap(false));
    });
    return () => {
      cancelAnimationFrame(outer);
      cancelAnimationFrame(inner);
    };
  }, [snap]);
  const [hidden, setHidden] = useState(false);
  // The selection the drawer was opened at; once the selection moves on, the drawer
  // reads as closed without an effect to close it.
  const [openedAt, setOpenedAt] = useState<string | null | undefined>(undefined);
  const drawerOpen = openedAt !== undefined && openedAt === detailsPath;
  const visible = narrow ? drawerOpen : !hidden;
  const handleToggle = useCallback(() => {
    if (narrow) setOpenedAt(drawerOpen ? undefined : detailsPath);
    else setHidden((prev) => !prev);
  }, [narrow, drawerOpen, detailsPath]);
  // With the drawer open the details are a backdrop: a click there closes the
  // drawer instead of reaching what is under it.
  const handleDetailsClickCapture = useCallback((e: MouseEvent) => {
    e.stopPropagation();
    setOpenedAt(undefined);
  }, []);
  const backdrop = narrow && drawerOpen;
  // One button in one corner: beside the list's title while it shows, beside the
  // details title once it is hidden.
  const toggle = (
    <Button.Button
      variant="text"
      size="small"
      tooltip={`${visible ? "Hide" : "Show"} ${listTitle.toLowerCase()}`}
      aria-expanded={visible}
      onClick={handleToggle}
    >
      <Icon.Sidebar />
    </Button.Button>
  );
  return (
    <Flex.Box
      ref={ref}
      x
      grow
      empty
      className={CSS.cls(
        CSS.B("panes"),
        narrow && CSS.M("narrow"),
        snap && CSS.M("snap"),
        CSS.M(visible ? "list-open" : "list-closed"),
      )}
    >
      <Flex.Box y empty className={CSS.BE("panes", "list")}>
        <Header.Header>
          <Flex.Box x align="center" gap="small" className={CSS.BE("panes", "title")}>
            {toggle}
            <Header.Title weight={500} color={10}>
              {listTitle}
            </Header.Title>
          </Flex.Box>
        </Header.Header>
        {list}
      </Flex.Box>
      <Divider.Divider y />
      <Flex.Box
        y
        grow
        empty
        className={CSS.BE("panes", "details")}
        onClickCapture={backdrop ? handleDetailsClickCapture : undefined}
      >
        <DetailsHeader
          path={detailsPath ?? ""}
          disabled={detailsPath == null}
          start={visible ? undefined : toggle}
        >
          {title}
        </DetailsHeader>
        {children}
      </Flex.Box>
    </Flex.Box>
  );
};
