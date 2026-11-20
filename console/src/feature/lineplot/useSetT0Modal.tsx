// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Triggers as PTriggers } from "@synnaxlabs/lyra/triggers";
import type { TimeRange } from "@synnaxlabs/x";
import { useState } from "react";

import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

export interface SetT0Params {
  /** The name of the range. */
  name: string;
  /** The time range of the range: the T that an offset like "T+12.5s" counts from. */
  timeRange: TimeRange;
  /** The current T0. */
  value: number;
}

const SetT0 = ({
  name,
  timeRange,
  value: initial,
  close,
}: Modals.ContentProps<SetT0Params, number>) => {
  const [value, setValue] = useState(initial);
  return (
    <Modal.Frame>
      <Modal.Header icon={<Icon.Time />}>{`Set T0 of ${name}`}</Modal.Header>
      <Modal.Body>
        <Input.Item
          label="T0"
          helpText="A time, or an offset from the start such as T+12.5s"
        >
          <Input.DateTime
            value={value}
            onChange={setValue}
            anchors={{ parent: timeRange.numeric }}
          />
        </Input.Item>
      </Modal.Body>
      <Modal.Footer>
        <Nav.Bar.End x align="center">
          <Button.Button
            variant="outlined"
            onClick={() => close()}
            triggerIndicator={PTriggers.ESCAPE}
          >
            Cancel
          </Button.Button>
          <Button.Button
            status="success"
            variant="filled"
            onClick={() => close(value)}
            trigger={Triggers.SAVE}
            triggerIndicator
          >
            Save
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

/** Opens a modal that asks for the T0 of a range on a line plot. */
export const useSetT0Modal = Modals.createPrompt<number, SetT0Params>(SetT0);
