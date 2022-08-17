package cesium

import (
	"context"
)

func createSync(ctx context.Context, c Create, segments *[]Segment) error {
	req, res, err := c.Stream(ctx)
	if err != nil {
		return err
	}
	req <- CreateRequest{Segments: *segments}
	close(req)
	resErr := (<-res).Error
	if resErr != nil {
		err = resErr
	}
	return err
}

func retrieveSync(ctx context.Context, r Retrieve, segments *[]Segment) error {
	resV := make(chan RetrieveResponse)
	var err error
	go func() {
		err = r.Stream(ctx, resV)
	}()
	for res := range resV {
		*segments = append(*segments, res.Segments...)
	}
	return err
}

func syncExec(ctx context.Context, query interface{}, seg *[]Segment) error {
	switch query.(type) {
	case Create:
		return createSync(ctx, query.(Create), seg)
	case Retrieve:
		return retrieveSync(ctx, query.(Retrieve), seg)
	}
	panic("only create and retrieve queries can be run synchronously")
}
