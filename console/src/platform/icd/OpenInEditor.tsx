// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import type { FC, ReactElement } from "react";

import { FLAGS } from "@/flags";
import { Panel } from "@/platform/panel";
import { Session } from "@/session";

export interface OpenInEditorProps {
  icdKey: icd.Key;
  /** The message the editor selects. */
  message: icd.MessageKey;
}

const EditorButton = ({ icdKey, message }: OpenInEditorProps): ReactElement => {
  const openTab = Panel.useOpenTab();
  const dispatch = Session.useDispatch();
  const handleClick = () => {
    dispatch(Session.ICD.setSelected({ key: icdKey, selected: [message] }));
    openTab({ variant: "resource", resource: icd.resourceID(icdKey) });
  };
  return (
    <Button.Button
      variant="text"
      size="small"
      tooltip="Open in the ICD editor"
      onClick={handleClick}
    >
      <Icon.Library />
      Open in editor
    </Button.Button>
  );
};

/** Opens the ICD editor at a message. Renders nothing while ICDs are behind a flag. */
export const OpenInEditor: FC<OpenInEditorProps> = FLAGS.icds
  ? EditorButton
  : () => null;
