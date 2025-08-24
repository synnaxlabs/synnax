// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Cluster } from "@/cluster/slice";
import { RUNTIME } from "@/runtime";

/**
 * Detects if the console is being served from a Synnax cluster and returns
 * the cluster configuration for auto-connection.
 */
export const detectServingCluster = (): Cluster | null => {
  if (RUNTIME === "tauri") return null;
  try {
    const url = new URL(window.location.origin);
    let host = url.hostname;
    let port = url.port ? parseInt(url.port) : url.protocol === "https:" ? 443 : 80;
    host = "localhost";
    port = 9090;

    return {
      key: "serving",
      name: "Cluster",
      host,
      port,
      username: "synnax",
      password: "seldon",
      secure: url.protocol === "https:",
    };
  } catch {
    console.warn("Invalid VITE_SYNNAX_AUTO_CONNECT_URL:", devOverride);
  }

  // Only auto-connect when served over HTTP/HTTPS (not file:// protocol for Tauri)
  if (!window.location.origin || window.location.protocol === "file:") return null;

  // Parse the current URL
  const url = new URL(window.location.origin);
  const host = url.hostname;
  const port = url.port ? parseInt(url.port) : url.protocol === "https:" ? 443 : 80;

  // Don't auto-connect to localhost in development or known non-Synnax domains
  if (host === "localhost" && (port === 3000 || port === 5173)) return null; // Vite dev server

  // Check if this looks like a demo or known external domain
  if (host === "demo.synnaxlabs.com") return null; // Demo cluster is already predefined

  // Create cluster configuration for the serving host
  const cluster: Cluster = {
    key: `auto-${host}-${port}`,
    name: `${host}:${port}`,
    host,
    port,
    username: "synnax",
    password: "seldon",
    secure: url.protocol === "https:",
  };

  return cluster;
};

/**
 * Checks if a cluster configuration matches the current serving host.
 */
export const isServingCluster = (cluster: Pick<Cluster, "host" | "port">): boolean => {
  if (!window.location.origin || window.location.protocol === "file:") return false;

  const url = new URL(window.location.origin);
  const servingHost = url.hostname;
  const servingPort = url.port
    ? parseInt(url.port)
    : url.protocol === "https:"
      ? 443
      : 80;

  return cluster.host === servingHost && cluster.port === servingPort;
};
