// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Button } from "@synnaxlabs/pluto";
import { binary } from "@synnaxlabs/x";

import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";
import { Link } from "@/link";

interface UseButtonProps {
  getConfig: () => unknown;
  getName: () => string;
  key?: task.Key;
}

export const useButtons = ({ getConfig, getName, key }: UseButtonProps) => {
  const copy = useCopyToClipboard();
  const handleCopyJSONConfig = () => {
    copy(
      binary.JSON_CODEC.encodeString(getConfig()),
      `JSON Configuration for ${getName()}`,
    );
  };
  return (
    <Align.Space direction="x" empty>
      {key != null && (
        <Link.CopyToolbarButton name={getName()} ontologyID={task.ontologyID(key)} />
      )}
      <Button.Icon
        tooltip={"Copy JSON Configuration"}
        tooltipLocation="left"
        variant="text"
        onClick={handleCopyJSONConfig}
      >
        <Icon.JSON style={{ color: "var(--pluto-gray-l7)" }} />
      </Button.Icon>
    </Align.Space>
  );
};
