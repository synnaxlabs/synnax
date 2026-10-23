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
        d="M11.646 4.104C11.882 3.868 12.117 3.868 12.353 4.104L19.896 11.647C20.132 11.883 20.132 12.118 19.896 12.354L12.353 19.897C12.117 20.133 11.882 20.133 11.646 19.897L4.103 12.354C3.867 12.118 3.867 11.883 4.103 11.647ZM6.791 11.788C6.65 11.929 6.65 12.071 6.791 12.212L11.788 17.209C11.929 17.35 12.071 17.35 12.212 17.209L17.209 12.212C17.35 12.071 17.35 11.929 17.209 11.788L12.212 6.791C12.071 6.65 11.929 6.65 11.788 6.791Z"
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M9.5 3.75C9.5 2.369 10.619 1.25 12 1.25C13.381 1.25 14.5 2.369 14.5 3.75C14.5 5.131 13.381 6.25 12 6.25C10.619 6.25 9.5 5.131 9.5 3.75Z"
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M17.75 12C17.75 10.619 18.869 9.5 20.25 9.5C21.631 9.5 22.75 10.619 22.75 12C22.75 13.381 21.631 14.5 20.25 14.5C18.869 14.5 17.75 13.381 17.75 12Z"
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M9.5 20.25C9.5 18.869 10.619 17.75 12 17.75C13.381 17.75 14.5 18.869 14.5 20.25C14.5 21.631 13.381 22.75 12 22.75C10.619 22.75 9.5 21.631 9.5 20.25Z"
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M1.25 12C1.25 10.619 2.369 9.5 3.75 9.5C5.131 9.5 6.25 10.619 6.25 12C6.25 13.381 5.131 14.5 3.75 14.5C2.369 14.5 1.25 13.381 1.25 12Z"
      />
    </svg>
  ),
  "bridge",
);
