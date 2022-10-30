"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.synnaxPropsSchema = void 0;
const freighter_1 = require("@synnaxlabs/freighter");
const zod_1 = require("zod");
const auth_1 = __importDefault(require("./auth"));
const creator_1 = __importDefault(require("./channel/creator"));
const registry_1 = __importDefault(require("./channel/registry"));
const retriever_1 = __importDefault(require("./channel/retriever"));
const connectivity_1 = __importDefault(require("./connectivity"));
const client_1 = __importDefault(require("./segment/client"));
const telem_1 = require("./telem");
const transport_1 = __importDefault(require("./transport"));
const client_2 = __importDefault(require("./ontology/client"));
const channel_1 = require("./channel");
exports.synnaxPropsSchema = zod_1.z.object({
    host: zod_1.z.string().min(1),
    port: zod_1.z.number().or(zod_1.z.string()),
    username: zod_1.z.string().optional(),
    password: zod_1.z.string().optional(),
    connectivityPollFrequency: zod_1.z.instanceof(telem_1.TimeSpan).optional(),
});
/**
 * Client to perform operations against a Synnax cluster.
 *
 * @property channel - Channel client for creating and retrieving channels.
 * @property data - Data client for reading and writing telemetry.
 */
class Synnax {
    /**
     * @param props.host - Hostname of a node in the cluster.
     * @param props.port - Port of the node in the cluster.
     * @param props.username - Username for authentication. Not required if the
     *   cluster is insecure.
     * @param props.password - Password for authentication. Not required if the
     *   cluster is insecure.
     */
    constructor({ host, port, username, password, connectivityPollFrequency, }) {
        this.transport = new transport_1.default(new freighter_1.URL({ host, port: Number(port) }));
        if (username && password) {
            this.auth = new auth_1.default(this.transport.httpFactory, {
                username,
                password,
            });
            this.transport.use(this.auth.middleware());
        }
        const chRetriever = new retriever_1.default(this.transport);
        const chCreator = new creator_1.default(this.transport);
        const chRegistry = new registry_1.default(chRetriever);
        this.data = new client_1.default(this.transport, chRegistry);
        this.channel = new channel_1.ChannelClient(this.data, chRetriever, chCreator);
        this.connectivity = new connectivity_1.default(this.transport.getClient(), connectivityPollFrequency);
        this.ontology = new client_2.default(this.transport);
    }
    close() {
        this.connectivity.stopChecking();
    }
}
exports.default = Synnax;
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiY2xpZW50LmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiLi4vLi4vLi4vc3JjL2xpYi9jbGllbnQudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6Ijs7Ozs7O0FBQUEscURBQTRDO0FBQzVDLDZCQUF3QjtBQUV4QixrREFBMEM7QUFDMUMsZ0VBQStDO0FBQy9DLGtFQUEwQztBQUMxQyxvRUFBbUQ7QUFDbkQsa0VBQWdEO0FBQ2hELDhEQUE2QztBQUM3QyxtQ0FBbUM7QUFDbkMsNERBQW9DO0FBQ3BDLCtEQUErQztBQUMvQyx1Q0FBMEM7QUFFN0IsUUFBQSxpQkFBaUIsR0FBRyxPQUFDLENBQUMsTUFBTSxDQUFDO0lBQ3hDLElBQUksRUFBRSxPQUFDLENBQUMsTUFBTSxFQUFFLENBQUMsR0FBRyxDQUFDLENBQUMsQ0FBQztJQUN2QixJQUFJLEVBQUUsT0FBQyxDQUFDLE1BQU0sRUFBRSxDQUFDLEVBQUUsQ0FBQyxPQUFDLENBQUMsTUFBTSxFQUFFLENBQUM7SUFDL0IsUUFBUSxFQUFFLE9BQUMsQ0FBQyxNQUFNLEVBQUUsQ0FBQyxRQUFRLEVBQUU7SUFDL0IsUUFBUSxFQUFFLE9BQUMsQ0FBQyxNQUFNLEVBQUUsQ0FBQyxRQUFRLEVBQUU7SUFDL0IseUJBQXlCLEVBQUUsT0FBQyxDQUFDLFVBQVUsQ0FBQyxnQkFBUSxDQUFDLENBQUMsUUFBUSxFQUFFO0NBQzdELENBQUMsQ0FBQztBQUlIOzs7OztHQUtHO0FBQ0gsTUFBcUIsTUFBTTtJQVF6Qjs7Ozs7OztPQU9HO0lBQ0gsWUFBWSxFQUNWLElBQUksRUFDSixJQUFJLEVBQ0osUUFBUSxFQUNSLFFBQVEsRUFDUix5QkFBeUIsR0FDYjtRQUNaLElBQUksQ0FBQyxTQUFTLEdBQUcsSUFBSSxtQkFBUyxDQUFDLElBQUksZUFBRyxDQUFDLEVBQUUsSUFBSSxFQUFFLElBQUksRUFBRSxNQUFNLENBQUMsSUFBSSxDQUFDLEVBQUUsQ0FBQyxDQUFDLENBQUM7UUFDdEUsSUFBSSxRQUFRLElBQUksUUFBUSxFQUFFO1lBQ3hCLElBQUksQ0FBQyxJQUFJLEdBQUcsSUFBSSxjQUFvQixDQUFDLElBQUksQ0FBQyxTQUFTLENBQUMsV0FBVyxFQUFFO2dCQUMvRCxRQUFRO2dCQUNSLFFBQVE7YUFDVCxDQUFDLENBQUM7WUFDSCxJQUFJLENBQUMsU0FBUyxDQUFDLEdBQUcsQ0FBQyxJQUFJLENBQUMsSUFBSSxDQUFDLFVBQVUsRUFBRSxDQUFDLENBQUM7U0FDNUM7UUFDRCxNQUFNLFdBQVcsR0FBRyxJQUFJLG1CQUFnQixDQUFDLElBQUksQ0FBQyxTQUFTLENBQUMsQ0FBQztRQUN6RCxNQUFNLFNBQVMsR0FBRyxJQUFJLGlCQUFjLENBQUMsSUFBSSxDQUFDLFNBQVMsQ0FBQyxDQUFDO1FBQ3JELE1BQU0sVUFBVSxHQUFHLElBQUksa0JBQVEsQ0FBQyxXQUFXLENBQUMsQ0FBQztRQUM3QyxJQUFJLENBQUMsSUFBSSxHQUFHLElBQUksZ0JBQWEsQ0FBQyxJQUFJLENBQUMsU0FBUyxFQUFFLFVBQVUsQ0FBQyxDQUFDO1FBQzFELElBQUksQ0FBQyxPQUFPLEdBQUcsSUFBSSx1QkFBYSxDQUFDLElBQUksQ0FBQyxJQUFJLEVBQUUsV0FBVyxFQUFFLFNBQVMsQ0FBQyxDQUFDO1FBQ3BFLElBQUksQ0FBQyxZQUFZLEdBQUcsSUFBSSxzQkFBa0IsQ0FDeEMsSUFBSSxDQUFDLFNBQVMsQ0FBQyxTQUFTLEVBQUUsRUFDMUIseUJBQXlCLENBQzFCLENBQUM7UUFDRixJQUFJLENBQUMsUUFBUSxHQUFHLElBQUksZ0JBQWMsQ0FBQyxJQUFJLENBQUMsU0FBUyxDQUFDLENBQUM7SUFDckQsQ0FBQztJQUVELEtBQUs7UUFDSCxJQUFJLENBQUMsWUFBWSxDQUFDLFlBQVksRUFBRSxDQUFDO0lBQ25DLENBQUM7Q0FDRjtBQTlDRCx5QkE4Q0MifQ==