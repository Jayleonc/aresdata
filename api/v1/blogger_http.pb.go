// Code generated by protoc-gen-go-http. DO NOT EDIT.
// versions:
// - protoc-gen-go-http v2.8.4
// - protoc             v3.20.3
// source: v1/blogger.proto

package v1

import (
	context "context"
	http "github.com/go-kratos/kratos/v2/transport/http"
	binding "github.com/go-kratos/kratos/v2/transport/http/binding"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the kratos package it is being compiled against.
var _ = new(context.Context)
var _ = binding.EncodeURL

const _ = http.SupportPackageIsVersion1

const OperationBloggerServiceGetBlogger = "/BloggerService/GetBlogger"
const OperationBloggerServiceListBloggers = "/BloggerService/ListBloggers"

type BloggerServiceHTTPServer interface {
	// GetBlogger 查询单个视频博主信息
	GetBlogger(context.Context, *BloggerQueryRequest) (*BloggerQueryResponse, error)
	// ListBloggers 分页查询视频博主信息
	ListBloggers(context.Context, *ListBloggersRequest) (*ListBloggersResponse, error)
}

func RegisterBloggerServiceHTTPServer(s *http.Server, srv BloggerServiceHTTPServer) {
	r := s.Route("/")
	r.POST("/v1/bloggers/detail", _BloggerService_GetBlogger0_HTTP_Handler(srv))
	r.POST("/v1/bloggers/list", _BloggerService_ListBloggers0_HTTP_Handler(srv))
}

func _BloggerService_GetBlogger0_HTTP_Handler(srv BloggerServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in BloggerQueryRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationBloggerServiceGetBlogger)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.GetBlogger(ctx, req.(*BloggerQueryRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*BloggerQueryResponse)
		return ctx.Result(200, reply)
	}
}

func _BloggerService_ListBloggers0_HTTP_Handler(srv BloggerServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in ListBloggersRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationBloggerServiceListBloggers)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.ListBloggers(ctx, req.(*ListBloggersRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*ListBloggersResponse)
		return ctx.Result(200, reply)
	}
}

type BloggerServiceHTTPClient interface {
	GetBlogger(ctx context.Context, req *BloggerQueryRequest, opts ...http.CallOption) (rsp *BloggerQueryResponse, err error)
	ListBloggers(ctx context.Context, req *ListBloggersRequest, opts ...http.CallOption) (rsp *ListBloggersResponse, err error)
}

type BloggerServiceHTTPClientImpl struct {
	cc *http.Client
}

func NewBloggerServiceHTTPClient(client *http.Client) BloggerServiceHTTPClient {
	return &BloggerServiceHTTPClientImpl{client}
}

func (c *BloggerServiceHTTPClientImpl) GetBlogger(ctx context.Context, in *BloggerQueryRequest, opts ...http.CallOption) (*BloggerQueryResponse, error) {
	var out BloggerQueryResponse
	pattern := "/v1/bloggers/detail"
	path := binding.EncodeURL(pattern, in, false)
	opts = append(opts, http.Operation(OperationBloggerServiceGetBlogger))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "POST", path, in, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *BloggerServiceHTTPClientImpl) ListBloggers(ctx context.Context, in *ListBloggersRequest, opts ...http.CallOption) (*ListBloggersResponse, error) {
	var out ListBloggersResponse
	pattern := "/v1/bloggers/list"
	path := binding.EncodeURL(pattern, in, false)
	opts = append(opts, http.Operation(OperationBloggerServiceListBloggers))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "POST", path, in, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
