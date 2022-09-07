package stream

import (
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/stream"
	sv1 "github.com/arya-analytics/delta/pkg/distribution/transport/grpc/gen/proto/go/stream/v1"
	"github.com/arya-analytics/freighter/fgrpc"
	"github.com/arya-analytics/x/telem"
)

var (
	_ fgrpc.Translator[stream.WriteRequest, *sv1.WriteRequest] = (*writeRequestTranslator)(nil)
	_ fgrpc.Translator[stream.ReadRequest, *sv1.ReadRequest]   = (*readRequestTranslator)(nil)
	_ fgrpc.Translator[stream.ReadResponse, *sv1.ReadResponse] = (*readResponseTranslator)(nil)
)

type writeRequestTranslator struct{}

func (w writeRequestTranslator) Backward(req *sv1.WriteRequest) (stream.WriteRequest, error) {
	samples, err := translateSamplesBackward(req.Samples)
	return stream.WriteRequest{Samples: samples}, err
}

func (w writeRequestTranslator) Forward(req stream.WriteRequest) (*sv1.WriteRequest, error) {
	return &sv1.WriteRequest{Samples: translateSamplesForward(req.Samples)}, nil
}

type readRequestTranslator struct{}

func (w readRequestTranslator) Backward(req *sv1.ReadRequest) (stream.ReadRequest, error) {
	keys, err := channel.ParseKeys(req.Keys)
	return stream.ReadRequest{Keys: keys}, err
}

func (w readRequestTranslator) Forward(req stream.ReadRequest) (*sv1.ReadRequest, error) {
	return &sv1.ReadRequest{Keys: req.Keys.Strings()}, nil
}

type readResponseTranslator struct{}

func (w readResponseTranslator) Backward(resp *sv1.ReadResponse) (stream.ReadResponse, error) {
	samples, err := translateSamplesBackward(resp.Samples)
	return stream.ReadResponse{Samples: samples}, err
}

func (w readResponseTranslator) Forward(resp stream.ReadResponse) (*sv1.ReadResponse, error) {
	return &sv1.ReadResponse{Samples: translateSamplesForward(resp.Samples)}, nil
}

func translateSamplesBackward(samples []*sv1.Sample) ([]stream.Sample, error) {
	s := make([]stream.Sample, len(samples))
	for i, sample := range samples {
		key, err := channel.ParseKey(sample.ChannelKey)
		if err != nil {
			return nil, err
		}
		s[i] = stream.Sample{
			ChannelKey: key,
			Stamp:      telem.TimeStamp(sample.Stamp),
			Value:      sample.Value,
		}
	}
	return s, nil
}

func translateSamplesForward(samples []stream.Sample) []*sv1.Sample {
	s := make([]*sv1.Sample, len(samples))
	for i, sample := range samples {
		s[i] = &sv1.Sample{
			ChannelKey: sample.ChannelKey.String(),
			Stamp:      int64(sample.Stamp),
			Value:      sample.Value,
		}
	}
	return s
}
