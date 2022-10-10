type URLProps = {
  host: string;
  port: number;
  protocol?: string;
  pathPrefix?: string;
  params?: string;
};

/**
 * URL is a simple class for building and extending URLs.
 */
export default class URL {
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
  constructor({ host, port, protocol = '', pathPrefix = '' }: URLProps) {
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
      host: props.host || this.host,
      port: props.port || this.port,
      protocol: props.protocol || this.protocol,
      pathPrefix: props.pathPrefix || this.path,
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
  stringify(): string {
    return removeTrailingSlash(
      `${this.protocol}://${this.host}:${this.port}/${this.path}`
    );
  }
}

// joinPath joins the two paths, ensuring there is a single slash between them.
const joinPaths = (...paths: string[]): string => {
  return paths.map(formatPath).join('');
};

const formatPath = (path: string): string => {
  if (!path.endsWith('/')) path += '/';
  if (path.startsWith('/')) path = path.slice(1);
  return path;
};

const removeTrailingSlash = (path: string): string => {
  if (path.endsWith('/')) path = path.slice(0, -1);
  return path;
};
