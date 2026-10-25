// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useMemo } from "react";

interface CodePanelProps {
  html: string;
  activeLines: number[];
  className?: string;
}

const LINE_TAG = '<span class="line">';
const ACTIVE_LINE_TAG = '<span class="line" data-active="true">';

export const CodePanel = ({
  html,
  activeLines,
  className,
}: CodePanelProps): ReactElement => {
  const processedHtml = useMemo(
    () =>
      html
        .split(LINE_TAG)
        .map((part, line) =>
          line === 0
            ? part
            : `${activeLines.includes(line) ? ACTIVE_LINE_TAG : LINE_TAG}${part}`,
        )
        .join(""),
    [html, activeLines],
  );

  return (
    <div
      className={`code-panel ${className ?? ""}`}
      dangerouslySetInnerHTML={{ __html: processedHtml }}
    />
  );
};
