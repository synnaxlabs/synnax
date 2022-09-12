export default class Endpoint {
    protocol;
    host;
    port;
    pathPrefix;
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
// joinPath joins the two paths, ensuring there is a single slash between them.
const joinPath = (a, b) => {
    // Remove any leading slashes from b.
    b = b.replace(/^\/+/, '');
    // Remove any trailing slashes from a.
    a = a.replace(/\/+$/, '');
    // Join the two paths with a single slash.
    return `${a}/${b}`;
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZW5kcG9pbnQuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyIuLi8uLi8uLi9zcmMvbGliL2VuZHBvaW50LnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQU9BLE1BQU0sQ0FBQyxPQUFPLE9BQU8sUUFBUTtJQUMzQixRQUFRLENBQVM7SUFDakIsSUFBSSxDQUFTO0lBQ2IsSUFBSSxDQUFTO0lBQ2IsVUFBVSxDQUFTO0lBRW5CLFlBQVksRUFBRSxJQUFJLEVBQUUsSUFBSSxFQUFFLFFBQVEsR0FBRyxFQUFFLEVBQUUsVUFBVSxHQUFHLEVBQUUsRUFBaUI7UUFDdkUsSUFBSSxDQUFDLFFBQVEsR0FBRyxRQUFRLENBQUM7UUFDekIsSUFBSSxDQUFDLElBQUksR0FBRyxJQUFJLENBQUM7UUFDakIsSUFBSSxDQUFDLElBQUksR0FBRyxJQUFJLENBQUM7UUFDakIsSUFBSSxDQUFDLFVBQVUsR0FBRyxVQUFVLENBQUM7SUFDL0IsQ0FBQztJQUVELEtBQUssQ0FBQyxFQUNKLElBQUksRUFDSixRQUFRLEdBQUcsRUFBRSxHQUlkO1FBQ0MsT0FBTyxJQUFJLFFBQVEsQ0FBQztZQUNsQixJQUFJLEVBQUUsSUFBSSxDQUFDLElBQUk7WUFDZixJQUFJLEVBQUUsSUFBSSxDQUFDLElBQUk7WUFDZixRQUFRLEVBQUUsUUFBUSxJQUFJLElBQUksQ0FBQyxRQUFRO1lBQ25DLFVBQVUsRUFBRSxJQUFJLENBQUMsQ0FBQyxDQUFDLFFBQVEsQ0FBQyxJQUFJLENBQUMsVUFBVSxFQUFFLElBQUksQ0FBQyxDQUFDLENBQUMsQ0FBQyxJQUFJLENBQUMsVUFBVTtTQUNyRSxDQUFDLENBQUM7SUFDTCxDQUFDO0lBRUQsSUFBSSxDQUFDLElBQVk7UUFDZixPQUFPLFFBQVEsQ0FBQyxJQUFJLENBQUMsR0FBRyxFQUFFLEVBQUUsSUFBSSxDQUFDLENBQUM7SUFDcEMsQ0FBQztJQUVELEdBQUc7UUFDRCxPQUFPLFFBQVEsQ0FDYixHQUFHLElBQUksQ0FBQyxRQUFRLE1BQU0sSUFBSSxDQUFDLElBQUksSUFBSSxJQUFJLENBQUMsSUFBSSxFQUFFLEVBQzlDLElBQUksQ0FBQyxVQUFVLENBQ2hCLENBQUM7SUFDSixDQUFDO0NBQ0Y7QUFFRCwrRUFBK0U7QUFDL0UsTUFBTSxRQUFRLEdBQUcsQ0FBQyxDQUFTLEVBQUUsQ0FBUyxFQUFVLEVBQUU7SUFDaEQscUNBQXFDO0lBQ3JDLENBQUMsR0FBRyxDQUFDLENBQUMsT0FBTyxDQUFDLE1BQU0sRUFBRSxFQUFFLENBQUMsQ0FBQztJQUMxQixzQ0FBc0M7SUFDdEMsQ0FBQyxHQUFHLENBQUMsQ0FBQyxPQUFPLENBQUMsTUFBTSxFQUFFLEVBQUUsQ0FBQyxDQUFDO0lBQzFCLDBDQUEwQztJQUMxQyxPQUFPLEdBQUcsQ0FBQyxJQUFJLENBQUMsRUFBRSxDQUFDO0FBQ3JCLENBQUMsQ0FBQyJ9