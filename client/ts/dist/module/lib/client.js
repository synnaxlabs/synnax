import { URL } from '@synnaxlabs/freighter';
import { z } from 'zod';
import AuthenticationClient from './auth';
import ChannelCreator from './channel/creator';
import Registry from './channel/registry';
import ChannelRetriever from './channel/retriever';
import ConnectivityClient from './connectivity';
import SegmentClient from './segment/client';
import { TimeSpan } from './telem';
import Transport from './transport';
import OntologyClient from './ontology/client';
import { ChannelClient } from './channel';
export const synnaxPropsSchema = z.object({
    host: z.string().min(1),
    port: z.number().or(z.string()),
    username: z.string().optional(),
    password: z.string().optional(),
    connectivityPollFrequency: z.instanceof(TimeSpan).optional(),
});
/**
 * Client to perform operations against a Synnax cluster.
 *
 * @property channel - Channel client for creating and retrieving channels.
 * @property data - Data client for reading and writing telemetry.
 */
export default class Synnax {
    transport;
    data;
    channel;
    auth;
    connectivity;
    ontology;
    /**
     * @param props.host - Hostname of a node in the cluster.
     * @param props.port - Port of the node in the cluster.
     * @param props.username - Username for authentication. Not required if the
     *   cluster is insecure.
     * @param props.password - Password for authentication. Not required if the
     *   cluster is insecure.
     */
    constructor({ host, port, username, password, connectivityPollFrequency, }) {
        this.transport = new Transport(new URL({ host, port: Number(port) }));
        if (username && password) {
            this.auth = new AuthenticationClient(this.transport.httpFactory, {
                username,
                password,
            });
            this.transport.use(this.auth.middleware());
        }
        const chRetriever = new ChannelRetriever(this.transport);
        const chCreator = new ChannelCreator(this.transport);
        const chRegistry = new Registry(chRetriever);
        this.data = new SegmentClient(this.transport, chRegistry);
        this.channel = new ChannelClient(this.data, chRetriever, chCreator);
        this.connectivity = new ConnectivityClient(this.transport.getClient(), connectivityPollFrequency);
        this.ontology = new OntologyClient(this.transport);
    }
    close() {
        this.connectivity.stopChecking();
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiY2xpZW50LmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiLi4vLi4vLi4vc3JjL2xpYi9jbGllbnQudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsT0FBTyxFQUFFLEdBQUcsRUFBRSxNQUFNLHVCQUF1QixDQUFDO0FBQzVDLE9BQU8sRUFBRSxDQUFDLEVBQUUsTUFBTSxLQUFLLENBQUM7QUFFeEIsT0FBTyxvQkFBb0IsTUFBTSxRQUFRLENBQUM7QUFDMUMsT0FBTyxjQUFjLE1BQU0sbUJBQW1CLENBQUM7QUFDL0MsT0FBTyxRQUFRLE1BQU0sb0JBQW9CLENBQUM7QUFDMUMsT0FBTyxnQkFBZ0IsTUFBTSxxQkFBcUIsQ0FBQztBQUNuRCxPQUFPLGtCQUFrQixNQUFNLGdCQUFnQixDQUFDO0FBQ2hELE9BQU8sYUFBYSxNQUFNLGtCQUFrQixDQUFDO0FBQzdDLE9BQU8sRUFBRSxRQUFRLEVBQUUsTUFBTSxTQUFTLENBQUM7QUFDbkMsT0FBTyxTQUFTLE1BQU0sYUFBYSxDQUFDO0FBQ3BDLE9BQU8sY0FBYyxNQUFNLG1CQUFtQixDQUFDO0FBQy9DLE9BQU8sRUFBRSxhQUFhLEVBQUUsTUFBTSxXQUFXLENBQUM7QUFFMUMsTUFBTSxDQUFDLE1BQU0saUJBQWlCLEdBQUcsQ0FBQyxDQUFDLE1BQU0sQ0FBQztJQUN4QyxJQUFJLEVBQUUsQ0FBQyxDQUFDLE1BQU0sRUFBRSxDQUFDLEdBQUcsQ0FBQyxDQUFDLENBQUM7SUFDdkIsSUFBSSxFQUFFLENBQUMsQ0FBQyxNQUFNLEVBQUUsQ0FBQyxFQUFFLENBQUMsQ0FBQyxDQUFDLE1BQU0sRUFBRSxDQUFDO0lBQy9CLFFBQVEsRUFBRSxDQUFDLENBQUMsTUFBTSxFQUFFLENBQUMsUUFBUSxFQUFFO0lBQy9CLFFBQVEsRUFBRSxDQUFDLENBQUMsTUFBTSxFQUFFLENBQUMsUUFBUSxFQUFFO0lBQy9CLHlCQUF5QixFQUFFLENBQUMsQ0FBQyxVQUFVLENBQUMsUUFBUSxDQUFDLENBQUMsUUFBUSxFQUFFO0NBQzdELENBQUMsQ0FBQztBQUlIOzs7OztHQUtHO0FBQ0gsTUFBTSxDQUFDLE9BQU8sT0FBTyxNQUFNO0lBQ2pCLFNBQVMsQ0FBWTtJQUM3QixJQUFJLENBQWdCO0lBQ3BCLE9BQU8sQ0FBZ0I7SUFDdkIsSUFBSSxDQUFtQztJQUN2QyxZQUFZLENBQXFCO0lBQ2pDLFFBQVEsQ0FBaUI7SUFFekI7Ozs7Ozs7T0FPRztJQUNILFlBQVksRUFDVixJQUFJLEVBQ0osSUFBSSxFQUNKLFFBQVEsRUFDUixRQUFRLEVBQ1IseUJBQXlCLEdBQ2I7UUFDWixJQUFJLENBQUMsU0FBUyxHQUFHLElBQUksU0FBUyxDQUFDLElBQUksR0FBRyxDQUFDLEVBQUUsSUFBSSxFQUFFLElBQUksRUFBRSxNQUFNLENBQUMsSUFBSSxDQUFDLEVBQUUsQ0FBQyxDQUFDLENBQUM7UUFDdEUsSUFBSSxRQUFRLElBQUksUUFBUSxFQUFFO1lBQ3hCLElBQUksQ0FBQyxJQUFJLEdBQUcsSUFBSSxvQkFBb0IsQ0FBQyxJQUFJLENBQUMsU0FBUyxDQUFDLFdBQVcsRUFBRTtnQkFDL0QsUUFBUTtnQkFDUixRQUFRO2FBQ1QsQ0FBQyxDQUFDO1lBQ0gsSUFBSSxDQUFDLFNBQVMsQ0FBQyxHQUFHLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQyxVQUFVLEVBQUUsQ0FBQyxDQUFDO1NBQzVDO1FBQ0QsTUFBTSxXQUFXLEdBQUcsSUFBSSxnQkFBZ0IsQ0FBQyxJQUFJLENBQUMsU0FBUyxDQUFDLENBQUM7UUFDekQsTUFBTSxTQUFTLEdBQUcsSUFBSSxjQUFjLENBQUMsSUFBSSxDQUFDLFNBQVMsQ0FBQyxDQUFDO1FBQ3JELE1BQU0sVUFBVSxHQUFHLElBQUksUUFBUSxDQUFDLFdBQVcsQ0FBQyxDQUFDO1FBQzdDLElBQUksQ0FBQyxJQUFJLEdBQUcsSUFBSSxhQUFhLENBQUMsSUFBSSxDQUFDLFNBQVMsRUFBRSxVQUFVLENBQUMsQ0FBQztRQUMxRCxJQUFJLENBQUMsT0FBTyxHQUFHLElBQUksYUFBYSxDQUFDLElBQUksQ0FBQyxJQUFJLEVBQUUsV0FBVyxFQUFFLFNBQVMsQ0FBQyxDQUFDO1FBQ3BFLElBQUksQ0FBQyxZQUFZLEdBQUcsSUFBSSxrQkFBa0IsQ0FDeEMsSUFBSSxDQUFDLFNBQVMsQ0FBQyxTQUFTLEVBQUUsRUFDMUIseUJBQXlCLENBQzFCLENBQUM7UUFDRixJQUFJLENBQUMsUUFBUSxHQUFHLElBQUksY0FBYyxDQUFDLElBQUksQ0FBQyxTQUFTLENBQUMsQ0FBQztJQUNyRCxDQUFDO0lBRUQsS0FBSztRQUNILElBQUksQ0FBQyxZQUFZLENBQUMsWUFBWSxFQUFFLENBQUM7SUFDbkMsQ0FBQztDQUNGIn0=