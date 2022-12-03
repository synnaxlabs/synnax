package channel

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	channelv1 "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/gen/proto/go/channel/v1"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/telem"
)

type createMessageTranslator struct{}

func (c createMessageTranslator) Forward(msg channel.CreateMessage) (*channelv1.CreateMessage, error) {
	tr := &channelv1.CreateMessage{}
	for _, ch := range msg.Channels {
		tr.Channels = append(tr.Channels, &channelv1.Channel{
			StorageKey:   int32(ch.StorageKey),
			Name:         ch.Name,
			NodeId:       int32(ch.NodeID),
			DataType:     string(ch.DataType),
			StorageIndex: int32(ch.LocalIndex),
			IsIndex:      ch.IsIndex,
			Rate:         float64(ch.Rate),
		})
	}
	return tr, nil
}

func (c createMessageTranslator) Backward(msg *channelv1.CreateMessage) (channel.CreateMessage, error) {
	var tr channel.CreateMessage
	for _, ch := range msg.Channels {
		tr.Channels = append(tr.Channels, channel.Channel{
			StorageKey: storage.ChannelKey(ch.StorageKey),
			Name:       ch.Name,
			NodeID:     dcore.NodeID(ch.NodeId),
			DataType:   telem.DataType(ch.DataType),
			LocalIndex: storage.ChannelKey(ch.StorageIndex),
			IsIndex:    ch.IsIndex,
			Rate:       telem.Rate(ch.Rate),
		})
	}
	return tr, nil
}
