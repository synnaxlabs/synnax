package writer

import (
	"context"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/confluence/plumber"
)

func newLocalWriter(
	ctx context.Context,
	keys channel.Keys,
	transient confluence.Inlet[error],
	cfg Config,
) (confluence.Segment[Request, Response], error) {
	w, err := cfg.TS.NewStreamWriter(keys.StorageKeys()...)
	if err != nil {
		return nil, err
	}
	pipe := plumber.New()
	plumber.SetSegment[cesium.WriteRequest, cesium.WriteResponse](pipe, "writer", w)
	reqT := newRequestTranslator(cfg.Resolver.HostID(), transient)
	resT := newResponseTranslator()
	plumber.SetSegment[Request, cesium.WriteRequest](pipe, "requestTranslator", reqT)
	plumber.SetSegment[cesium.WriteResponse, Response](pipe, "responseTranslator", resT)
	plumber.UnaryRouter[cesium.WriteRequest]{
		SourceTarget: "requestTranslator",
		SinkTarget:   "writer",
	}.MustRoute(pipe)
	plumber.UnaryRouter[cesium.WriteResponse]{
		SourceTarget: "writer",
		SinkTarget:   "responseTranslator",
	}.MustRoute(pipe)
	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}
	if err := seg.RouteInletTo("requestTranslator"); err != nil {
		panic(err)
	}
	if err := seg.RouteOutletFrom("responseTranslator"); err != nil {
		panic(err)
	}
	return seg, nil
}
