// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { wrapSVGIcon } from "@/icon/Icon";

export const Bridge = wrapSVGIcon(
  (props) => (
    <svg
      viewBox="0 0 24 24"
      xmlns="http://www.w3.org/2000/svg"
      stroke="currentColor"
      fill="currentColor"
      strokeWidth={0}
      {...props}
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M11.646 2.604C11.882 2.368 12.117 2.368 12.353 2.604L21.396 11.647C21.632 11.883 21.632 12.118 21.396 12.354L12.353 21.397C12.117 21.633 11.882 21.633 11.646 21.397L2.603 12.354C2.367 12.118 2.367 11.883 2.603 11.647ZM5.291 11.788C5.15 11.929 5.15 12.071 5.291 12.212L11.788 18.709C11.929 18.85 12.071 18.85 12.212 18.709L18.709 12.212C18.85 12.071 18.85 11.929 18.709 11.788L12.212 5.291C12.071 5.15 11.929 5.15 11.788 5.291Z"
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M9.5 2.25C9.5 0.869 10.619 -0.25 12 -0.25C13.381 -0.25 14.5 0.869 14.5 2.25C14.5 3.631 13.381 4.75 12 4.75C10.619 4.75 9.5 3.631 9.5 2.25Z"
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M19.25 12C19.25 10.619 20.369 9.5 21.75 9.5C23.131 9.5 24.25 10.619 24.25 12C24.25 13.381 23.131 14.5 21.75 14.5C20.369 14.5 19.25 13.381 19.25 12Z"
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M9.5 21.75C9.5 20.369 10.619 19.25 12 19.25C13.381 19.25 14.5 20.369 14.5 21.75C14.5 23.131 13.381 24.25 12 24.25C10.619 24.25 9.5 23.131 9.5 21.75Z"
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M-0.25 12C-0.25 10.619 0.869 9.5 2.25 9.5C3.631 9.5 4.75 10.619 4.75 12C4.75 13.381 3.631 14.5 2.25 14.5C0.869 14.5 -0.25 13.381 -0.25 12Z"
      />
    </svg>
  ),
  "bridge",
);
