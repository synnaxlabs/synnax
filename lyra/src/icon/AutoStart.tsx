// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { wrapSVGIcon } from "@/icon/Icon";

export const AutoStart = wrapSVGIcon(
  (props) => (
    <svg
      viewBox="0 0 24 24"
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      {...props}
    >
      <path d="M3.06 10.88a9 9 0 1 1 .56 4.5m-.56 5v-5h5" />
      <path d="M9.9 8.5v7l5.6 -3.5z" />
    </svg>
  ),
  "auto-start",
);

// As in Tabler's off icons, the lines stop short of the slash only above it.
export const AutoStartOff = wrapSVGIcon(
  (props) => (
    <svg
      viewBox="0 0 24 24"
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      {...props}
    >
      <path d="M3.06 10.88a9 9 0 0 1 2.594 -5.224" />
      <path d="M7.991 3.988a9 9 0 0 1 12.058 12.059" />
      <path d="M18.383 18.383a9 9 0 0 1 -14.763 -3.003m-.56 5v-5h5" />
      <path d="M13.346 13.346l-3.446 2.154v-5.6" />
      <path d="M3 3l18 18" />
    </svg>
  ),
  "auto-start-off",
);
