// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { Log as Base } from "@synnaxlabs/pluto";
import { location, TimeStamp } from "@synnaxlabs/x";
import { memo, type ReactElement, useState } from "react";

import { CSS } from "@/platform/css";
import { Vis } from "@/platform/vis";
import { Session } from "@/session";

const TIME_PLACEHOLDER = "2026-08-23 14:05:00";

const GoToTime = (): ReactElement => {
  const key = Base.useKey();
  const { timeZone } = Base.use({ key });
  const seek = Base.useSeek();
  const [visible, setVisible] = useState(false);
  const [text, setText] = useState("");
  const [invalid, setInvalid] = useState(false);
  const handleChange = (text: string): void => {
    setText(text);
    setInvalid(false);
  };
  const commit = (): void => {
    const time = TimeStamp.parse(text, timeZone);
    if (time == null) return setInvalid(true);
    seek(time);
    setVisible(false);
  };
  return (
    <Dialog.Frame
      visible={visible}
      onVisibleChange={setVisible}
      location={location.BOTTOM_LEFT}
    >
      <Dialog.Trigger
        hideCaret
        size="small"
        aria-label="Go to a time"
        tooltipLocation={location.BOTTOM_LEFT}
        tooltip={
          <Text.Text level="small" color={11}>
            Go to a time
          </Text.Text>
        }
      >
        <Icon.Time />
      </Dialog.Trigger>
      <Dialog.Dialog bordered borderColor={7}>
        <Input.Text
          autoFocus
          size="small"
          aria-label="Time"
          value={text}
          onChange={handleChange}
          onKeyDown={(e) => {
            if (e.key === "Enter") commit();
          }}
          placeholder={TIME_PLACEHOLDER}
          status={invalid ? "error" : undefined}
          tooltip={invalid ? "Not a time" : undefined}
        />
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};

export const Controls = memo((): ReactElement => {
  const key = Base.useKey();
  const hold = Session.Log.useSelectHold();
  const dispatch = Session.useDispatch();
  const handleHoldChange = (hold: boolean): void => {
    dispatch(Session.Log.setHold({ key, hold }));
  };
  return (
    <Vis.Controls>
      <GoToTime />
      <Button.Toggle
        value={hold}
        onChange={handleHoldChange}
        size="small"
        aria-label="Pause scrolling"
        className={CSS.cls(hold && Vis.CONTROLS_PINNED_CLASS)}
        tooltipLocation={location.BOTTOM_LEFT}
        tooltip={
          <Text.Text level="small" color={11}>
            {`${hold ? "Resume" : "Pause"} scrolling`}
            <Triggers.Text trigger={Base.PAUSE_TRIGGER} level="small" />
          </Text.Text>
        }
      >
        {hold ? <Icon.Play /> : <Icon.Pause />}
      </Button.Toggle>
    </Vis.Controls>
  );
});
Controls.displayName = "Controls";
