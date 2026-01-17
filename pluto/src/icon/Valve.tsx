// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { wrapSVGIcon } from "@/icon/Icon";

export const Valve = wrapSVGIcon(
  (props) => (
    <svg
      viewBox="0 0 16 9"
      xmlns="http://www.w3.org/2000/svg"
      stroke="currentColor"
      fill="currentColor"
      {...props}
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M0.19043 2.4044V7.13293C0.19043 7.88056 0.980882 8.3638 1.6463 8.02297L5.51219 6.04289C5.33639 5.67219 5.23805 5.25763 5.23805 4.8201C5.23805 4.34998 5.35159 3.90638 5.55275 3.51522L1.64631 1.51436C0.980883 1.17353 0.19043 1.65677 0.19043 2.4044ZM10.6398 6.12077C10.8396 5.73065 10.9523 5.28854 10.9523 4.8201C10.9523 4.31901 10.8233 3.84804 10.5967 3.4386L14.3536 1.51436C15.019 1.17353 15.8095 1.65677 15.8095 2.4044V7.13293C15.8095 7.88056 15.019 8.3638 14.3536 8.02297L10.6398 6.12077Z"
      />
      <circle cx="8.09518" cy="4.82016" r="2.35714" fill="none" />
    </svg>
  ),
  "valve",
);
