"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
class Endpoint {
    constructor({ host, port, protocol = '', pathPrefix = '' }) {
        this.protocol = protocol;
        this.host = host;
        this.port = port;
        this.pathPrefix = pathPrefix;
    }
    child({ path, protocol = '', }) {
        return new Endpoint({
            host: this.host,
            port: this.port,
            protocol: protocol || this.protocol,
            pathPrefix: path ? joinPath(this.pathPrefix, path) : this.pathPrefix,
        });
    }
    path(path) {
        return joinPath(this.uri(), path);
    }
    uri() {
        return joinPath(`${this.protocol}://${this.host}:${this.port}`, this.pathPrefix);
    }
}
exports.default = Endpoint;
// joinPath joins the two paths, ensuring there is a single slash between them.
const joinPath = (a, b) => {
    // Remove any leading slashes from b.
    b = b.replace(/^\/+/, '');
    // Remove any trailing slashes from a.
    a = a.replace(/\/+$/, '');
    // Join the two paths with a single slash.
    return `${a}/${b}`;
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZW5kcG9pbnQuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyIuLi8uLi8uLi9zcmMvbGliL2VuZHBvaW50LnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiI7O0FBT0EsTUFBcUIsUUFBUTtJQU0zQixZQUFZLEVBQUUsSUFBSSxFQUFFLElBQUksRUFBRSxRQUFRLEdBQUcsRUFBRSxFQUFFLFVBQVUsR0FBRyxFQUFFLEVBQWlCO1FBQ3ZFLElBQUksQ0FBQyxRQUFRLEdBQUcsUUFBUSxDQUFDO1FBQ3pCLElBQUksQ0FBQyxJQUFJLEdBQUcsSUFBSSxDQUFDO1FBQ2pCLElBQUksQ0FBQyxJQUFJLEdBQUcsSUFBSSxDQUFDO1FBQ2pCLElBQUksQ0FBQyxVQUFVLEdBQUcsVUFBVSxDQUFDO0lBQy9CLENBQUM7SUFFRCxLQUFLLENBQUMsRUFDSixJQUFJLEVBQ0osUUFBUSxHQUFHLEVBQUUsR0FJZDtRQUNDLE9BQU8sSUFBSSxRQUFRLENBQUM7WUFDbEIsSUFBSSxFQUFFLElBQUksQ0FBQyxJQUFJO1lBQ2YsSUFBSSxFQUFFLElBQUksQ0FBQyxJQUFJO1lBQ2YsUUFBUSxFQUFFLFFBQVEsSUFBSSxJQUFJLENBQUMsUUFBUTtZQUNuQyxVQUFVLEVBQUUsSUFBSSxDQUFDLENBQUMsQ0FBQyxRQUFRLENBQUMsSUFBSSxDQUFDLFVBQVUsRUFBRSxJQUFJLENBQUMsQ0FBQyxDQUFDLENBQUMsSUFBSSxDQUFDLFVBQVU7U0FDckUsQ0FBQyxDQUFDO0lBQ0wsQ0FBQztJQUVELElBQUksQ0FBQyxJQUFZO1FBQ2YsT0FBTyxRQUFRLENBQUMsSUFBSSxDQUFDLEdBQUcsRUFBRSxFQUFFLElBQUksQ0FBQyxDQUFDO0lBQ3BDLENBQUM7SUFFRCxHQUFHO1FBQ0QsT0FBTyxRQUFRLENBQ2IsR0FBRyxJQUFJLENBQUMsUUFBUSxNQUFNLElBQUksQ0FBQyxJQUFJLElBQUksSUFBSSxDQUFDLElBQUksRUFBRSxFQUM5QyxJQUFJLENBQUMsVUFBVSxDQUNoQixDQUFDO0lBQ0osQ0FBQztDQUNGO0FBdENELDJCQXNDQztBQUVELCtFQUErRTtBQUMvRSxNQUFNLFFBQVEsR0FBRyxDQUFDLENBQVMsRUFBRSxDQUFTLEVBQVUsRUFBRTtJQUNoRCxxQ0FBcUM7SUFDckMsQ0FBQyxHQUFHLENBQUMsQ0FBQyxPQUFPLENBQUMsTUFBTSxFQUFFLEVBQUUsQ0FBQyxDQUFDO0lBQzFCLHNDQUFzQztJQUN0QyxDQUFDLEdBQUcsQ0FBQyxDQUFDLE9BQU8sQ0FBQyxNQUFNLEVBQUUsRUFBRSxDQUFDLENBQUM7SUFDMUIsMENBQTBDO0lBQzFDLE9BQU8sR0FBRyxDQUFDLElBQUksQ0FBQyxFQUFFLENBQUM7QUFDckIsQ0FBQyxDQUFDIn0=