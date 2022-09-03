package iterator

import (
	"context"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	distribcore "github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
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
	plumber.SetSegment[cesium.IterateRequest, cesium.IterateResponse](pipe, "iterator", iter)
	plumber.SetSegment[Request, cesium.IterateRequest](
		pipe,
		"requestTranslator",
		newCesiumRequestTranslator(),
	)
	plumber.SetSegment[cesium.IterateResponse, Response](
		pipe,
		"responseTranslator",
		newCesiumResponseTranslator(cfg.Resolver.HostID()),
	)
	plumber.UnaryRouter[cesium.IterateRequest]{
		SourceTarget: "requestTranslator",
		SinkTarget:   "iterator",
	}.MustRoute(pipe)
	plumber.UnaryRouter[cesium.IterateResponse]{
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
	confluence.LinearTransform[cesium.IterateResponse, Response]
}

func newCesiumResponseTranslator(host distribcore.NodeID) confluence.Segment[cesium.IterateResponse, Response] {
	wrapper := &core.StorageWrapper{Host: host}
	ts := &storageResponseTranslator{wrapper: wrapper}
	ts.LinearTransform.Transform = ts.translate
	return ts
}

func (te *storageResponseTranslator) translate(
	ctx context.Context,
	res cesium.IterateResponse,
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
	confluence.LinearTransform[Request, cesium.IterateRequest]
}

func newCesiumRequestTranslator() confluence.Segment[Request, cesium.IterateRequest] {
	rq := &storageRequestTranslator{}
	rq.LinearTransform.Transform = rq.translate
	return rq
}

func (te *storageRequestTranslator) translate(
	ctx context.Context,
	req Request,
) (cesium.IterateRequest, bool, error) {
	return cesium.IterateRequest{
		Command: cesium.IteratorCommand(req.Command),
		Span:    req.Span,
		Range:   req.Range,
		Stamp:   req.Stamp,
	}, true, nil
}
