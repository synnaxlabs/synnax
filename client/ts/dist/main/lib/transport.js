"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const freighter_1 = require("@synnaxlabs/freighter");
class Transport {
    constructor(url) {
        this.url = url.child('/api/v1/');
        this.httpFactory = new freighter_1.HTTPClientFactory(this.url, new freighter_1.JSONEncoderDecoder());
        this.streamClient = new freighter_1.WebSocketClient(new freighter_1.JSONEncoderDecoder(), this.url);
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
exports.default = Transport;
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoidHJhbnNwb3J0LmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiLi4vLi4vLi4vc3JjL2xpYi90cmFuc3BvcnQudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6Ijs7QUFBQSxxREFRK0I7QUFFL0IsTUFBcUIsU0FBUztJQUs1QixZQUFZLEdBQVE7UUFDbEIsSUFBSSxDQUFDLEdBQUcsR0FBRyxHQUFHLENBQUMsS0FBSyxDQUFDLFVBQVUsQ0FBQyxDQUFDO1FBQ2pDLElBQUksQ0FBQyxXQUFXLEdBQUcsSUFBSSw2QkFBaUIsQ0FDdEMsSUFBSSxDQUFDLEdBQUcsRUFDUixJQUFJLDhCQUFrQixFQUFFLENBQ3pCLENBQUM7UUFDRixJQUFJLENBQUMsWUFBWSxHQUFHLElBQUksMkJBQWUsQ0FBQyxJQUFJLDhCQUFrQixFQUFFLEVBQUUsSUFBSSxDQUFDLEdBQUcsQ0FBQyxDQUFDO0lBQzlFLENBQUM7SUFFRCxTQUFTO1FBQ1AsT0FBTyxJQUFJLENBQUMsV0FBVyxDQUFDLFNBQVMsRUFBRSxDQUFDO0lBQ3RDLENBQUM7SUFFRCxVQUFVO1FBQ1IsT0FBTyxJQUFJLENBQUMsV0FBVyxDQUFDLFVBQVUsRUFBRSxDQUFDO0lBQ3ZDLENBQUM7SUFFRCxHQUFHLENBQUMsR0FBRyxVQUF3QjtRQUM3QixJQUFJLENBQUMsV0FBVyxDQUFDLEdBQUcsQ0FBQyxHQUFHLFVBQVUsQ0FBQyxDQUFDO1FBQ3BDLElBQUksQ0FBQyxZQUFZLENBQUMsR0FBRyxDQUFDLEdBQUcsVUFBVSxDQUFDLENBQUM7SUFDdkMsQ0FBQztDQUNGO0FBMUJELDRCQTBCQyJ9