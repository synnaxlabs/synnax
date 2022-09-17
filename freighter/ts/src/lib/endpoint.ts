type EndpointProps = {
  host: string;
  port: number;
  protocol?: string;
  pathPrefix?: string;
};

export default class Endpoint {
  protocol: string;
  host: string;
  port: number;
  pathPrefix: string;

  constructor({ host, port, protocol = '', pathPrefix = '' }: EndpointProps) {
    this.protocol = protocol;
    this.host = host;
    this.port = port;
    this.pathPrefix = pathPrefix;
  }

  child({
    path,
    protocol = '',
  }: {
    path?: string;
    protocol?: string;
  }): Endpoint {
    return new Endpoint({
      host: this.host,
      port: this.port,
      protocol: protocol || this.protocol,
      pathPrefix: path ? joinPath(this.pathPrefix, path) : this.pathPrefix,
    });
  }

  path(path: string): string {
    return joinPath(this.uri(), path);
  }

  uri(): string {
    return joinPath(
      `${this.protocol}://${this.host}:${this.port}`,
      this.pathPrefix
    );
  }
}

// joinPath joins the two paths, ensuring there is a single slash between them.
const joinPath = (a: string, b: string): string => {
  // Remove any leading slashes from b.
  b = b.replace(/^\/+/, '');
  // Remove any trailing slashes from a.
  a = a.replace(/\/+$/, '');
  // Join the two paths with a single slash.
  return `${a}/${b}`;
};
