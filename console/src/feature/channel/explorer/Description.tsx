// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { useResize } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Channel } from "@synnaxlabs/pluto";
import {
  type KeyboardEvent,
  type ReactElement,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";

import { Channel as PlatformChannel } from "@/platform/channel";
import { CSS } from "@/platform/css";

const CLASS = CSS.BE("channel-inspector", "description");

interface ClampedProps {
  value: string;
  /** Opens the editor when the user activates the text. Omit it for a reader. */
  onEdit?: () => void;
  /** Focuses the text when it mounts. */
  autoFocus: boolean;
}

/**
 * The description, cut to a few lines with a toggle when it runs longer. An empty one
 * invites an editor to add it.
 */
const Clamped = ({ value, onEdit, autoFocus }: ClampedProps): ReactElement => {
  const empty = value.length === 0;
  const ref = useRef<HTMLParagraphElement | null>(null);
  const [expanded, setExpanded] = useState(false);
  const [overflows, setOverflows] = useState(false);
  const measure = (el: HTMLElement): void => {
    if (!expanded) setOverflows(el.scrollHeight > el.clientHeight);
  };
  const resizeRef = useResize<HTMLParagraphElement>((_, el) => measure(el));
  useLayoutEffect(() => {
    if (ref.current != null) measure(ref.current);
  }, [value, expanded]);
  useEffect(() => {
    if (autoFocus) ref.current?.focus();
  }, []);
  return (
    <>
      <Text.Text
        ref={(el: HTMLParagraphElement | null) => {
          ref.current = el;
          resizeRef(el);
        }}
        level="small"
        color={9}
        className={CSS.cls(
          CSS.BE("channel-inspector", "text"),
          onEdit != null && CSS.M("editable"),
          expanded && CSS.M("expanded"),
          empty && CSS.M("empty"),
        )}
        {...(onEdit != null && {
          role: "button",
          tabIndex: 0,
          onClick: onEdit,
          onKeyDown: (e: KeyboardEvent<HTMLParagraphElement>) => {
            if (e.key !== "Enter" && e.key !== " ") return;
            e.preventDefault();
            onEdit();
          },
        })}
      >
        {empty ? (
          <>
            <Icon.Add />
            Add a description
          </>
        ) : (
          value
        )}
      </Text.Text>
      {(overflows || expanded) && (
        <Button.Button
          variant="text"
          size="small"
          className={CSS.BE("channel-inspector", "more")}
          aria-expanded={expanded}
          onClick={() => setExpanded((e) => !e)}
        >
          {expanded ? "Show less" : "Show more"}
        </Button.Button>
      )}
    </>
  );
};

export interface DescriptionProps {
  /** The key of the channel. */
  channelKey: channel.Key;
  /** The stored description of the channel. */
  value: string;
}

/**
 * The description of a channel. A user who may update the channel edits it in place:
 * blur or Cmd+Enter saves, and Escape cancels. A refused save opens the editor again
 * with the user's text. Any other user reads it.
 */
export const Description = ({
  channelKey,
  value,
}: DescriptionProps): ReactElement | null => {
  const editable = Access.useUpdateGranted(channel.resourceID(channelKey));
  const [draft, setDraft] = useState<string | null>(null);
  const [focusText, setFocusText] = useState(false);
  // An edit ends once, though Escape and Cmd+Enter blur the field after they end it.
  const editing = useRef(false);
  const edit = (text: string): void => {
    editing.current = true;
    setDraft(text);
  };
  const { update } = Channel.useSetDescription({
    afterFailure: ({ data }) => edit(data.description),
  });
  const finish = (save: boolean, keyboard: boolean): void => {
    if (!editing.current || draft == null) return;
    editing.current = false;
    setDraft(null);
    setFocusText(keyboard);
    const next = draft.trim();
    if (save && next !== value) update({ key: channelKey, description: next });
  };
  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>): void => {
    if (e.key === "Escape") finish(false, true);
    else if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      finish(true, true);
    }
  };
  if (draft != null)
    return (
      <Input.Text
        area
        autoFocus
        className={CSS.cls(CLASS, CSS.M("editing"))}
        value={draft}
        onChange={setDraft}
        onBlur={() => finish(true, false)}
        onKeyDown={handleKeyDown}
        onFocus={(e) => {
          const el = e.currentTarget;
          el.setSelectionRange(el.value.length, el.value.length);
          el.scrollTop = el.scrollHeight;
        }}
        maxLength={PlatformChannel.DESCRIPTION_MAX_LENGTH}
        placeholder="Add a description"
        aria-label="Channel description"
        size="small"
      />
    );
  if (!editable && value.length === 0) return null;
  return (
    <Flex.Box y empty className={CLASS}>
      <Clamped
        value={value}
        onEdit={editable ? () => edit(value) : undefined}
        autoFocus={focusText}
      />
    </Flex.Box>
  );
};
