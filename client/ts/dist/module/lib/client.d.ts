import { z } from 'zod';
import AuthenticationClient from './auth';
import ConnectivityClient from './connectivity';
import SegmentClient from './segment/client';
import { TimeSpan } from './telem';
import OntologyClient from './ontology/client';
import { ChannelClient } from './channel';
export declare const synnaxPropsSchema: z.ZodObject<{
    host: z.ZodString;
    port: z.ZodUnion<[z.ZodNumber, z.ZodString]>;
    username: z.ZodOptional<z.ZodString>;
    password: z.ZodOptional<z.ZodString>;
    connectivityPollFrequency: z.ZodOptional<z.ZodType<TimeSpan, z.ZodTypeDef, TimeSpan>>;
}, "strip", z.ZodTypeAny, {
    username?: string | undefined;
    password?: string | undefined;
    connectivityPollFrequency?: TimeSpan | undefined;
    host: string;
    port: string | number;
}, {
    username?: string | undefined;
    password?: string | undefined;
    connectivityPollFrequency?: TimeSpan | undefined;
    host: string;
    port: string | number;
}>;
export declare type SynnaxProps = z.infer<typeof synnaxPropsSchema>;
/**
 * Client to perform operations against a Synnax cluster.
 *
 * @property channel - Channel client for creating and retrieving channels.
 * @property data - Data client for reading and writing telemetry.
 */
export default class Synnax {
    private transport;
    data: SegmentClient;
    channel: ChannelClient;
    auth: AuthenticationClient | undefined;
    connectivity: ConnectivityClient;
    ontology: OntologyClient;
    /**
     * @param props.host - Hostname of a node in the cluster.
     * @param props.port - Port of the node in the cluster.
     * @param props.username - Username for authentication. Not required if the
     *   cluster is insecure.
     * @param props.password - Password for authentication. Not required if the
     *   cluster is insecure.
     */
    constructor({ host, port, username, password, connectivityPollFrequency, }: SynnaxProps);
    close(): void;
}
