import { HTTPClientFactory, JSONEncoderDecoder, WebSocketClient, } from '@synnaxlabs/freighter';
export default class Transport {
    url;
    httpFactory;
    streamClient;
    constructor(url) {
        this.url = url.child('/api/v1/');
        this.httpFactory = new HTTPClientFactory(this.url, new JSONEncoderDecoder());
        this.streamClient = new WebSocketClient(new JSONEncoderDecoder(), this.url);
    }
    getClient() {
        return this.httpFactory.getClient();
    }
    postClient() {
        return this.httpFactory.postClient();
    }
    use(...middleware) {
        this.httpFactory.use(...middleware);
        this.streamClient.use(...middleware);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoidHJhbnNwb3J0LmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiLi4vLi4vLi4vc3JjL2xpYi90cmFuc3BvcnQudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsT0FBTyxFQUNMLGlCQUFpQixFQUNqQixrQkFBa0IsRUFLbEIsZUFBZSxHQUNoQixNQUFNLHVCQUF1QixDQUFDO0FBRS9CLE1BQU0sQ0FBQyxPQUFPLE9BQU8sU0FBUztJQUM1QixHQUFHLENBQU07SUFDVCxXQUFXLENBQW9CO0lBQy9CLFlBQVksQ0FBZTtJQUUzQixZQUFZLEdBQVE7UUFDbEIsSUFBSSxDQUFDLEdBQUcsR0FBRyxHQUFHLENBQUMsS0FBSyxDQUFDLFVBQVUsQ0FBQyxDQUFDO1FBQ2pDLElBQUksQ0FBQyxXQUFXLEdBQUcsSUFBSSxpQkFBaUIsQ0FDdEMsSUFBSSxDQUFDLEdBQUcsRUFDUixJQUFJLGtCQUFrQixFQUFFLENBQ3pCLENBQUM7UUFDRixJQUFJLENBQUMsWUFBWSxHQUFHLElBQUksZUFBZSxDQUFDLElBQUksa0JBQWtCLEVBQUUsRUFBRSxJQUFJLENBQUMsR0FBRyxDQUFDLENBQUM7SUFDOUUsQ0FBQztJQUVELFNBQVM7UUFDUCxPQUFPLElBQUksQ0FBQyxXQUFXLENBQUMsU0FBUyxFQUFFLENBQUM7SUFDdEMsQ0FBQztJQUVELFVBQVU7UUFDUixPQUFPLElBQUksQ0FBQyxXQUFXLENBQUMsVUFBVSxFQUFFLENBQUM7SUFDdkMsQ0FBQztJQUVELEdBQUcsQ0FBQyxHQUFHLFVBQXdCO1FBQzdCLElBQUksQ0FBQyxXQUFXLENBQUMsR0FBRyxDQUFDLEdBQUcsVUFBVSxDQUFDLENBQUM7UUFDcEMsSUFBSSxDQUFDLFlBQVksQ0FBQyxHQUFHLENBQUMsR0FBRyxVQUFVLENBQUMsQ0FBQztJQUN2QyxDQUFDO0NBQ0YifQ==