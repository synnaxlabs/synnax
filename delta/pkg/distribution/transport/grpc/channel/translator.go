package channel

import (
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	distribcore "github.com/arya-analytics/delta/pkg/distribution/core"
	channelv1 "github.com/arya-analytics/delta/pkg/distribution/transport/grpc/gen/proto/go/channel/v1"
	"github.com/arya-analytics/x/telem"
)

type createMessageTranslator struct{}

func (c createMessageTranslator) Forward(msg channel.CreateMessage) (*channelv1.CreateMessage, error) {
	tr := &channelv1.CreateMessage{}
	for _, ch := range msg.Channels {
		tr.Channels = append(tr.Channels, &channelv1.Channel{
			Name:   ch.Name,
			NodeId: int32(ch.NodeID),
			Key:    int32(ch.Channel.Key),
			DataType: &channelv1.DataType{
				Key:     ch.DataType.Key,
				Density: int32(ch.DataType.Density),
			},
		})
	}
	return tr, nil
}

func (c createMessageTranslator) Backward(msg *channelv1.CreateMessage) (channel.CreateMessage, error) {
	var tr channel.CreateMessage
	for _, ch := range msg.Channels {
		tr.Channels = append(tr.Channels, channel.Channel{
			Name:   ch.Name,
			NodeID: distribcore.NodeID(ch.NodeId),
			DataType: telem.DataType{
				Key:     ch.DataType.Key,
				Density: telem.Density(ch.DataType.Density),
			},
			Channel: cesium.Channel{
				Key:     cesium.ChannelKey(ch.Key),
				Density: telem.Density(ch.DataType.Density),
			},
		})
	}
	return tr, nil
}
