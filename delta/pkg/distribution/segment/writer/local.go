package writer

import (
	"context"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/confluence/plumber"
)

func newLocalWriter(
	ctx context.Context,
	host core.NodeID,
	db cesium.DB,
	keys channel.Keys,
	transient confluence.Inlet[error],
) (confluence.Segment[Request, Response], error) {
	requests, responses, err := db.NewCreate().WhereChannels(keys.StorageKeys()...).Stream(ctx)
	if err != nil {
		return nil, err
	}
	reqT := newRequestTranslator(host, transient)
	reqT.OutTo(confluence.NewInlet[cesium.CreateRequest](requests))
	resT := newResponseTranslator()
	resT.InFrom(confluence.NewOutlet[cesium.CreateResponse](responses))
	pipe := plumber.New()
	plumber.SetSegment[Request, cesium.CreateRequest](pipe, "requestTranslator", reqT)
	plumber.SetSegment[cesium.CreateResponse, Response](pipe, "responseTranslator", resT)
	seg := &plumber.Segment[Request, Response]{Pipeline: pipe, NoAcquireForOutlets: true}
	if err := seg.RouteInletTo("requestTranslator"); err != nil {
		panic(err)
	}
	if err := seg.RouteOutletFrom("responseTranslator"); err != nil {
		panic(err)
	}
	return seg, nil
}
