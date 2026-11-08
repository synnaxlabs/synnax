// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { wrapSVGIcon } from "@/icon/Icon";

export const New = wrapSVGIcon(
  (props) => (
    <svg
      viewBox="0 0 24 24"
      xmlns="http://www.w3.org/2000/svg"
      stroke="currentColor"
      fill="currentColor"
      strokeWidth={0}
      {...props}
    >
      <circle cx="12" cy="12" r="11" fillOpacity={0.24} />
      <path d="M11 7.5H13V11H16.5V13H13V16.5H11V13H7.5V11H11Z" />
    </svg>
  ),
  "new",
);
