// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { type ReactElement, useCallback, useState } from "react";

import { type VizTab, VizTabs } from "@/components/common/VizTabs";

type Language = "python" | "typescript" | "cpp";

type Operation = "stream" | "write" | "read";

/** The highlighted HTML of each operation's sample in each language. */
export type Samples = Record<Language, Record<Operation, string>>;

interface SDKShowcaseProps {
  samples: Samples;
}

const LANG_TABS: Array<VizTab & { key: Language }> = [
  { key: "python", title: "Python", icon: Icon.Python },
  { key: "typescript", title: "TypeScript", icon: Icon.TypeScript },
  { key: "cpp", title: "C++", icon: Icon.CPlusPlus },
];

const OPERATIONS: Array<{ key: Operation; label: string }> = [
  { key: "stream", label: "Stream" },
  { key: "write", label: "Write" },
  { key: "read", label: "Read" },
];

export const SDKShowcase = ({ samples }: SDKShowcaseProps): ReactElement => {
  const [activeLang, setActiveLang] = useState(0);

  const handleLangClick = useCallback(
    (i: number) => {
      if (i !== activeLang) setActiveLang(i);
    },
    [activeLang],
  );

  return (
    <div className="sdks-showcase">
      <VizTabs tabs={LANG_TABS} active={activeLang} onSelect={handleLangClick} />
      <div className="sdks-panels">
        {OPERATIONS.map(({ key, label }) => (
          <div key={key} className="sdks-panel">
            <span className="sdks-panel-label">{label}</span>
            <div
              className="code-panel"
              dangerouslySetInnerHTML={{
                __html: samples[LANG_TABS[activeLang].key][key],
              }}
            />
          </div>
        ))}
      </div>
    </div>
  );
};
