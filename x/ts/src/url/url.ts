// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

interface URLProps {
  host: string;
  port: number;
  protocol?: string;
  pathPrefix?: string;
  params?: string;
}

/** @returns the paths joined with a single slash */
const joinPaths = (...paths: string[]): string => paths.map(formatPath).join("");

/** ensures that a path is correctly formatted for joining */
const formatPath = (path: string): string => {
  if (!path.endsWith("/")) path += "/";
  if (path.startsWith("/")) path = path.slice(1);
  return path;
};

/** removes the trailing slash from a path */
const removeTrailingSlash = (path: string): string =>
  path.endsWith("/") ? path.slice(0, -1) : path;

/**
 * Builds a query string from a record.
 * @param record - The record to build the query string from. If the record is null,
 * an empty string is returned.
 * @returns the query string.
 */
export const buildQueryString = (
  request: Record<string, string>,
  prefix: string = "",
): string => {
  if (request === null) return "";
  return `?${Object.entries(request)
    .filter(([, value]) => {
      if (value === undefined || value === null) return false;
      if (Array.isArray(value)) return value.length > 0;
      return true;
    })

    .map(([key, value]) => `${prefix}${key}=${value}`)
    .join("&")}`;
};

/**
 * URL is a simple class for building and extending URLs.
 */
export class URL {
  protocol: string;
  host: string;
  port: number;
  path: string;

  /**
   * @param host - The hostname or IP address of the server.
   * @param port - The port number of the server.
   * @param protocol - The protocol to use for all requests. Defaults to "".
   * @param pathPrefix - A path prefix to use for all requests. Defaults to "".
   */
  constructor({ host, port, protocol = "", pathPrefix = "" }: URLProps) {
    this.protocol = protocol;
    this.host = host;
    this.port = port;
    this.path = formatPath(pathPrefix);
  }

  /**
   * Replaces creates a new URL with the specified properties replaced.
   * @param props - The properties to replace.
   * @returns a new URL.
   */
  replace(props: Partial<URLProps>): URL {
    return new URL({
      host: props.host ?? this.host,
      port: props.port ?? this.port,
      protocol: props.protocol ?? this.protocol,
      pathPrefix: props.pathPrefix ?? this.path,
    });
  }

  /**
   * Creates a new url with the given path appended to the current path.
   * @param path - the path to append to the URL.
   * @returns a new URL.
   */
  child(path: string): URL {
    return new URL({
      ...this,
      pathPrefix: joinPaths(this.path, path),
    });
  }

  /** @returns a string representation of the url */
  toString(): string {
    return removeTrailingSlash(
      `${this.protocol}://${this.host}:${this.port}/${this.path}`,
    );
  }

  static readonly UNKNOWN = new URL({ host: "unknown", port: 0 });
}
