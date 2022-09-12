declare type EndpointProps = {
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
    constructor({ host, port, protocol, pathPrefix }: EndpointProps);
    child({ path, protocol, }: {
        path?: string;
        protocol?: string;
    }): Endpoint;
    path(path: string): string;
    uri(): string;
}
export {};
