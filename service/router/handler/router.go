package handler

import (
	"context"
	"io"
	"time"

	"c-z.dev/go-micro/errors"
	"c-z.dev/go-micro/router"
	pb "c-z.dev/go-micro/router/service/proto"
)

// Router implements router handler
type Router struct {
	Router router.Router
}

// Lookup looks up routes in the routing table and returns them
func (r *Router) Lookup(ctx context.Context, req *pb.LookupRequest, resp *pb.LookupResponse) error {
	routes, err := r.Router.Lookup(router.QueryService(req.Query.Service))
	if err != nil {
		return errors.InternalServerError("go.micro.router", "failed to lookup routes: %v", err)
	}

	respRoutes := make([]*pb.Route, 0, len(routes))
	for _, route := range routes {
		respRoute := &pb.Route{
			Service: route.Service,
			Address: route.Address,
			Gateway: route.Gateway,
			Network: route.Network,
			Router:  route.Router,
			Link:    route.Link,
			Metric:  route.Metric,
		}
		respRoutes = append(respRoutes, respRoute)
	}

	resp.Routes = respRoutes

	return nil
}

// Advertise streams router advertisements
func (r *Router) Advertise(ctx context.Context, req *pb.Request, stream pb.Router_AdvertiseStream) error {
	advertChan, err := r.Router.Advertise()
	if err != nil {
		return errors.InternalServerError("go.micro.router", "failed to get adverts: %v", err)
	}

	for advert := range advertChan {
		var events []*pb.Event
		for _, event := range advert.Events {
			route := &pb.Route{
				Service: event.Route.Service,
				Address: event.Route.Address,
				Gateway: event.Route.Gateway,
				Network: event.Route.Network,
				Router:  event.Route.Router,
				Link:    event.Route.Link,
				Metric:  event.Route.Metric,
			}
			e := &pb.Event{
				Id:        event.Id,
				Type:      pb.EventType(event.Type),
				Timestamp: event.Timestamp.UnixNano(),
				Route:     route,
			}
			events = append(events, e)
		}

		advert := &pb.Advert{
			Id:        advert.Id,
			Type:      pb.AdvertType(advert.Type),
			Timestamp: advert.Timestamp.UnixNano(),
			Events:    events,
		}

		// send the advert
		err := stream.Send(advert)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errors.InternalServerError("go.micro.router", "error sending message %v", err)
		}
	}

	return nil
}

// Process processes advertisements
func (r *Router) Process(ctx context.Context, req *pb.Advert, rsp *pb.ProcessResponse) error {
	events := make([]*router.Event, len(req.Events))
	for i, event := range req.Events {
		route := router.Route{
			Service: event.Route.Service,
			Address: event.Route.Address,
			Gateway: event.Route.Gateway,
			Network: event.Route.Network,
			Router:  event.Route.Router,
			Link:    event.Route.Link,
			Metric:  event.Route.Metric,
		}

		events[i] = &router.Event{
			Id:        event.Id,
			Type:      router.EventType(event.Type),
			Timestamp: time.Unix(0, event.Timestamp),
			Route:     route,
		}
	}

	advert := &router.Advert{
		Id:        req.Id,
		Type:      router.AdvertType(req.Type),
		Timestamp: time.Unix(0, req.Timestamp),
		TTL:       time.Duration(req.Ttl),
		Events:    events,
	}

	if err := r.Router.Process(advert); err != nil {
		return errors.InternalServerError("go.micro.router", "error publishing advert: %v", err)
	}

	return nil
}

// Watch streans routing table events
func (r *Router) Watch(ctx context.Context, req *pb.WatchRequest, stream pb.Router_WatchStream) error {
	watcher, err := r.Router.Watch()
	if err != nil {
		return errors.InternalServerError("go.micro.router", "failed creating event watcher: %v", err)
	}
	defer watcher.Stop()
	defer stream.Close()

	for {
		event, err := watcher.Next()
		if err == router.ErrWatcherStopped {
			return errors.InternalServerError("go.micro.router", "watcher stopped")
		}

		if err != nil {
			return errors.InternalServerError("go.micro.router", "error watching events: %v", err)
		}

		route := &pb.Route{
			Service: event.Route.Service,
			Address: event.Route.Address,
			Gateway: event.Route.Gateway,
			Network: event.Route.Network,
			Router:  event.Route.Router,
			Link:    event.Route.Link,
			Metric:  event.Route.Metric,
		}

		tableEvent := &pb.Event{
			Id:        event.Id,
			Type:      pb.EventType(event.Type),
			Timestamp: event.Timestamp.UnixNano(),
			Route:     route,
		}

		if err := stream.Send(tableEvent); err != nil {
			return err
		}
	}
}
