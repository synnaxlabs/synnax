package iterator

import (
	"context"
	"github.com/arya-analytics/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	distribcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment/core"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/confluence/plumber"
	"github.com/samber/lo"
)

func newLocalIterator(keys channel.Keys, cfg Config) (confluence.Segment[Request, Response], error) {
	iter, err := cfg.TS.NewStreamIterator(cfg.TimeRange, keys.StorageKeys()...)
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
		newCesiumResponseTranslator(cfg.Resolver.HostID()),
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

type storageResponseTranslator struct {
	wrapper *core.StorageWrapper
	confluence.LinearTransform[cesium.IteratorResponse, Response]
}

func newCesiumResponseTranslator(host distribcore.NodeID) confluence.Segment[cesium.IteratorResponse, Response] {
	wrapper := &core.StorageWrapper{Host: host}
	ts := &storageResponseTranslator{wrapper: wrapper}
	ts.LinearTransform.Transform = ts.translate
	return ts
}

func (te *storageResponseTranslator) translate(
	ctx context.Context,
	res cesium.IteratorResponse,
) (Response, bool, error) {
	return Response{
		Ack:      res.Ack,
		Variant:  ResponseVariant(res.Variant),
		Counter:  res.Counter,
		NodeID:   te.wrapper.Host,
		Error:    res.Err,
		Command:  Command(res.Command),
		Segments: te.wrapper.Wrap(res.Segments),
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
		Range:   req.Range,
		Stamp:   req.Stamp,
	}, true, nil
}
