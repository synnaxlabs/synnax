package iterator

import (
	"context"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	distribcore "github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/confluence/plumber"
	"github.com/arya-analytics/x/errutil"
	"github.com/arya-analytics/x/telem"
	"github.com/cockroachdb/errors"
)

func newLocalIterator(
	db cesium.DB,
	host distribcore.NodeID,
	rng telem.TimeRange,
	keys channel.Keys,
) (confluence.Segment[Request, Response], error) {

	iter := db.NewRetrieve().WhereTimeRange(rng).WhereChannels(keys.StorageKeys()...).Iterate()
	if iter.Error() != nil {
		return nil, errors.Wrap(iter.Error(), "[segment.iterator] - server failed to open cesium iterator")
	}

	pipe := plumber.New()

	// cesiumRes receives segments from the iterator.
	plumber.SetSource[cesium.RetrieveResponse](pipe, "iterator", iter)

	// executor executes requests as method calls on the iterator. Pipes
	// synchronous acknowledgements out to the response pipeline.
	te := newRequestExecutor(host, iter)
	plumber.SetSegment[Request, Response](pipe, "executor", te)

	// translator translates cesium res from the iterator source into
	// res transportable over the network.
	ts := newCesiumResponseTranslator(host)
	plumber.SetSegment[cesium.RetrieveResponse, Response](pipe, "translator", ts)

	c := errutil.NewCatch()

	c.Exec(plumber.UnaryRouter[cesium.RetrieveResponse]{
		SourceTarget: "iterator",
		SinkTarget:   "translator",
	}.PreRoute(pipe))

	if c.Error() != nil {
		panic(c.Error())
	}

	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}

	if err := seg.RouteInletTo("executor"); err != nil {
		panic(err)
	}

	if err := seg.RouteOutletFrom("translator", "executor"); err != nil {
		panic(err)
	}

	return seg, nil
}

type requestExecutor struct {
	host distribcore.NodeID
	iter cesium.StreamIterator
	confluence.LinearTransform[Request, Response]
}

func newRequestExecutor(
	host distribcore.NodeID,
	iter cesium.StreamIterator,
) confluence.Segment[Request, Response] {
	te := &requestExecutor{iter: iter, host: host}
	te.LinearTransform.Transform = te.execute
	return te
}

func (te *requestExecutor) execute(ctx context.Context, req Request) (Response, bool, error) {
	res := executeRequest(ctx, te.host, te.iter, req)
	// If we don't have a valid response, don't send it.
	return res, res.Variant != 0, nil
}

type cesiumResponseTranslator struct {
	wrapper *core.StorageWrapper
	confluence.LinearTransform[cesium.RetrieveResponse, Response]
}

func newCesiumResponseTranslator(host distribcore.NodeID) confluence.Segment[cesium.RetrieveResponse, Response] {
	wrapper := &core.StorageWrapper{Host: host}
	ts := &cesiumResponseTranslator{wrapper: wrapper}
	ts.LinearTransform.Transform = ts.translate
	return ts
}

func (te *cesiumResponseTranslator) translate(
	ctx context.Context,
	res cesium.RetrieveResponse,
) (Response, bool, error) {
	return Response{Variant: DataResponse, Segments: te.wrapper.Wrap(res.Segments)}, true, nil
}
