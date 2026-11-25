// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ReactElement, SVGProps } from "react";

import { CSS } from "@/css";
import { wrapMulticolor } from "@/icon/Colored";
import { wrapSVGIcon } from "@/icon/Icon";

// Marks for identity providers that no icon set carries. Each is one filled path that
// takes the text color; the even-odd rule cuts the inner shapes out.
const mark = (viewBox: string, d: string, name: string) =>
  wrapSVGIcon(
    ({ className, ...rest }: SVGProps<SVGSVGElement>): ReactElement => (
      <svg
        viewBox={viewBox}
        {...rest}
        className={CSS.cls(className, "logo")}
        fill="currentColor"
        strokeWidth={0}
        xmlns="http://www.w3.org/2000/svg"
      >
        <path d={d} fillRule="evenodd" />
      </svg>
    ),
    name,
  );

export const JumpCloud = mark(
  "0 0 24 24",
  "M7 19h10.5a4.5 4.5 0 0 0 .6-8.96A6 6 0 0 0 6.6 9.06 5 5 0 0 0 7 19z",
  "logo-jumpcloud",
);

export const OneLogin = mark(
  "0 0 24 24",
  "M12 2a10 10 0 1 1 0 20 10 10 0 0 1 0-20zM10.8 7h2.6v10h-2.6V9.9l-1.7.8-.8-1.9z",
  "logo-onelogin",
);

export const PingIdentity = mark(
  "0 0 24 24",
  "M5 2h14a3 3 0 0 1 3 3v14a3 3 0 0 1-3 3H5a3 3 0 0 1-3-3V5a3 3 0 0 1 3-3z" +
    "M8 6v12h2.6v-4H13a4 4 0 0 0 0-8zm2.6 2.4v3.2H13a1.6 1.6 0 0 0 0-3.2z",
  "logo-pingidentity",
);

export const Zitadel = mark(
  "0 0 24 24",
  "M12 2.5 22.5 20.5h-21zM12 8.4 6.6 17.7h10.8z",
  "logo-zitadel",
);

export const Duo = mark(
  "0 0 30 12",
  "M0 0h4.5a6 6 0 0 1 0 12H0zM11 0h3v6.2a1.5 1.5 0 0 0 3 0V0h3v6.2a4.5 4.5 0 0 1-9 0z" +
    "M25.5 1.5a4.5 4.5 0 1 1 0 9 4.5 4.5 0 0 1 0-9z",
  "logo-duo",
);

export const Google = wrapMulticolor(
  "0 0 48 48",
  [
    [
      "#EA4335",
      "M24 9.5c3.54 0 6.71 1.22 9.21 3.6l6.85-6.85C35.9 2.38 30.47 0 24 0 14.62 0 " +
        "6.51 5.38 2.56 13.22l7.98 6.19C12.43 13.72 17.74 9.5 24 9.5z",
    ],
    [
      "#4285F4",
      "M46.98 24.55c0-1.57-.15-3.09-.38-4.55H24v9.02h12.94c-.58 2.96-2.26 5.48-4.78 " +
        "7.18l7.73 6c4.51-4.18 7.09-10.36 7.09-17.65z",
    ],
    [
      "#FBBC04",
      "M10.53 28.59c-.48-1.45-.76-2.99-.76-4.59s.27-3.14.76-4.59l-7.98-6.19C.92 " +
        "16.46 0 20.12 0 24c0 3.88.92 7.54 2.56 10.78l7.97-6.19z",
    ],
    [
      "#34A853",
      "M24 48c6.48 0 11.93-2.13 15.89-5.81l-7.73-6c-2.15 1.45-4.92 2.3-8.16 2.3-6.26 " +
        "0-11.57-4.22-13.47-9.91l-7.98 6.19C6.51 42.62 14.62 48 24 48z",
    ],
  ],
  "logo-google",
);

export const Microsoft = wrapMulticolor(
  "0 0 23 23",
  [
    ["#F35325", "M1 1h10v10H1z"],
    ["#81BC06", "M12 1h10v10H12z"],
    ["#05A6F0", "M1 12h10v10H1z"],
    ["#FFBA08", "M12 12h10v10H12z"],
  ],
  "logo-microsoft",
);
