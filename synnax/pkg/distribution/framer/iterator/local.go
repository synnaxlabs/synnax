package iterator

import (
	"context"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	distribcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
)

func (s *Service) newGatewayIterator(cfg Config) (confluence.Segment[Request, Response], error) {
	iter, err := s.TS.NewStreamIterator(cesium.IteratorConfig{
		Bounds:   cfg.Bounds,
		Channels: cfg.Keys.Strings(),
	})
	if err != nil {
		return nil, err
	}
	pipe := plumber.New()
	plumber.SetSegment[cesium.IteratorRequest, cesium.IteratorResponse](pipe, "iterator", iter)
	plumber.SetSegment[Request, cesium.IteratorRequest](
		pipe,
		"requestTranslator",
		newCesiumRequestTranslator(),
	)
	plumber.SetSegment[cesium.IteratorResponse, Response](
		pipe,
		"responseTranslator",
		newCesiumResponseTranslator(s.HostResolver.HostID()),
	)
	plumber.UnaryRouter[cesium.IteratorRequest]{
		SourceTarget: "requestTranslator",
		SinkTarget:   "iterator",
	}.MustRoute(pipe)
	plumber.UnaryRouter[cesium.IteratorResponse]{
		SourceTarget: "iterator",
		SinkTarget:   "responseTranslator",
	}.MustRoute(pipe)
	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}
	lo.Must0(seg.RouteInletTo("requestTranslator"))
	lo.Must0(seg.RouteOutletFrom("responseTranslator"))
	return seg, nil
}

func newCesiumResponseTranslator(host distribcore.NodeID) confluence.Segment[cesium.IteratorResponse, Response] {
	wrapper := &core.StorageWrapper{Host: host}
	ts := &storageResponseTranslator{wrapper: wrapper}
	ts.LinearTransform.Transform = ts.translate
	return ts
}

type storageResponseTranslator struct {
	wrapper *core.StorageWrapper
	confluence.LinearTransform[cesium.IteratorResponse, Response]
}

func (te *storageResponseTranslator) translate(
	ctx context.Context,
	res cesium.IteratorResponse,
) (Response, bool, error) {
	return Response{
		Ack:     res.Ack,
		Variant: ResponseVariant(res.Variant),
		SeqNum:  res.SeqNum,
		NodeID:  te.wrapper.Host,
		Err:     res.Err,
		Command: Command(res.Command),
		Frame:   te.wrapper.Wrap(res.Frame),
	}, true, nil
}

type storageRequestTranslator struct {
	confluence.LinearTransform[Request, cesium.IteratorRequest]
}

func newCesiumRequestTranslator() confluence.Segment[Request, cesium.IteratorRequest] {
	rq := &storageRequestTranslator{}
	rq.LinearTransform.Transform = rq.translate
	return rq
}

func (te *storageRequestTranslator) translate(
	ctx context.Context,
	req Request,
) (cesium.IteratorRequest, bool, error) {
	return cesium.IteratorRequest{
		Command: cesium.IteratorCommand(req.Command),
		Span:    req.Span,
		Stamp:   req.Stamp,
	}, true, nil
}
