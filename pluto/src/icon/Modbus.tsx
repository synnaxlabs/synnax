// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSS } from "@/css";
import { wrapSVGIcon } from "@/icon/Icon";

export const Modbus = wrapSVGIcon(
  ({ className, ...rest }) => (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 52 60"
      className={CSS(className, "logo")}
      {...rest}
      stroke="currentColor"
      fill="currentColor"
    >
      <circle cx="6" cy="18" r="6" />
      <circle cx="26" cy="6" r="6" />
      <circle cx="46" cy="18" r="6" />
      <circle cx="46" cy="39" r="6" />
      <circle cx="26" cy="51" r="6" />
      <circle cx="6" cy="39" r="6" />
      <path d="M26 30L23.1132 35H28.8868L26 30ZM26.5 51V34.5H25.5V51H26.5Z" />
      <path d="M26 28L28.8868 23H23.1132L26 28ZM25.5 6V23.5H26.5V6H25.5Z" />
      <path d="M24 30L18.2369 29.6541L20.8189 34.8181L24 30ZM6.22361 39.4472L20.1987 32.4597L19.7515 31.5652L5.77639 38.5528L6.22361 39.4472Z" />
      <path d="M28 30L31.1811 34.8181L33.7631 29.6541L28 30ZM46.2236 38.5528L32.2485 31.5652L31.8013 32.4597L45.7764 39.4472L46.2236 38.5528Z" />
      <path d="M28 28L33.7727 28.0953L30.9689 23.0483L28 28ZM45.7572 17.5629L31.6909 25.3775L32.1765 26.2517L46.2428 18.4371L45.7572 17.5629Z" />
      <path d="M24 28L21.0311 23.0483L18.2273 28.0953L24 28ZM5.75718 18.4371L19.8235 26.2517L20.3091 25.3775L6.24282 17.5629L5.75718 18.4371Z" />{" "}
    </svg>
  ),
  "logo-modbus",
);
