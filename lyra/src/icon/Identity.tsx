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
