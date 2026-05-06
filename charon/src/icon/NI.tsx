// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSS } from "@/css";
import { wrapSVGIcon } from "@/icon/Icon";

export const NI = wrapSVGIcon(
  ({ className, ...rest }) => (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 78 51"
      {...rest}
      className={CSS(className, "logo")}
      stroke="currentColor"
      fill="currentColor"
    >
      <g clipPath="url(#clip0_327_656)">
        <path
          d="M17.3026 17.0048V51.0427H0V17.0048H17.3026ZM34.6051 -0.0140575C36.8587 -0.0327593 39.0939 0.392621 41.1831 1.23779C43.2723 2.08297 45.1746 3.33138 46.7813 4.91175C48.388 6.49211 49.6677 8.37348 50.5473 10.4484C51.4269 12.5234 51.8891 14.7512 51.9077 17.0048V51.0427H34.6051V17.0048H17.3026V-0.0140575H34.6051ZM77.8615 -0.0140575V51.0427C75.6074 51.0632 73.3714 50.6391 71.2813 49.7946C69.1913 48.9501 67.2883 47.7018 65.6812 46.1211C64.0741 44.5403 62.7945 42.6582 61.9156 40.5824C61.0366 38.5066 60.5756 36.2779 60.559 34.0238V-0.0140575H77.8615Z"
          fill="#03B584"
        />
      </g>
      <defs>
        <clipPath id="clip0_327_656">
          <rect width="77.8615" height="51" fill="white" />
        </clipPath>
      </defs>
    </svg>
  ),
  "logo-ni",
);
