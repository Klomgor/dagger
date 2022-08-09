package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

	"github.com/Khan/genqlient/graphql"
	"github.com/containerd/containerd/platforms"
	"github.com/dagger/cloak/api"
	"github.com/dagger/cloak/core"
	"github.com/dagger/cloak/router"
	"github.com/dagger/cloak/sdk/go/dagger"
	bkclient "github.com/moby/buildkit/client"
	bkgw "github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/solver/pb"
	"github.com/moby/buildkit/util/progress/progressui"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/sync/errgroup"

	_ "github.com/moby/buildkit/client/connhelper/dockercontainer" // import the docker connection driver
	"github.com/moby/buildkit/client/llb"
)

type StartOpts struct {
	Export *bkclient.ExportEntry
	// TODO: All these fields can in theory be more dynamic, added after Start is called, but that requires
	// varying levels of effort (secrets are easy, local dirs are hard unless we patch upstream buildkit)
	LocalDirs map[string]string
	Secrets   map[string]string
	DevServer int
}

type StartCallback func(ctx context.Context, localDirs map[string]dagger.FSID, secrets map[string]string) (dagger.FSID, error)

func Start(ctx context.Context, startOpts *StartOpts, fn StartCallback) error {
	opts := []bkclient.ClientOpt{
		bkclient.WithFailFast(),
		// bkclient.WithTracerProvider(otel.GetTracerProvider()),
	}

	// exp, err := detect.Exporter()
	// if err != nil {
	// 	return err
	// }

	// if td, ok := exp.(bkclient.TracerDelegate); ok {
	// 	opts = append(opts, bkclient.WithTracerDelegate(td))
	// }

	c, err := bkclient.New(ctx, "docker-container://dagger-buildkitd", opts...)
	if err != nil {
		return err
	}

	platform, err := detectPlatform(ctx, c)
	if err != nil {
		return err
	}

	ch := make(chan *bkclient.SolveStatus)

	var server api.Server
	solveOpts := bkclient.SolveOpt{
		Session: []session.Attachable{&server},
	}
	if startOpts != nil {
		if startOpts.Export != nil {
			solveOpts.Exports = []bkclient.ExportEntry{*startOpts.Export}
		}
		solveOpts.LocalDirs = startOpts.LocalDirs
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		var err error
		_, err = c.Build(ctx, solveOpts, "", func(ctx context.Context, gw bkgw.Client) (*bkgw.Result, error) {
			secrets := make(map[string]string)
			secretIDToKey := make(map[string]string)
			for k, v := range startOpts.Secrets {
				hashkey := sha256.Sum256([]byte(v))
				hashVal := hex.EncodeToString(hashkey[:])
				secrets[hashVal] = v
				secretIDToKey[k] = hashVal
			}

			router := router.New()
			if err := router.Add(core.New(router, gw, *platform)...); err != nil {
				return nil, err
			}

			// server = api.NewServer(gw, platform, secrets)

			ctx = withInMemoryAPIClient(ctx, router)
			// ctx = withGatewayClient(ctx, gw)
			// ctx = withPlatform(ctx, platform)

			cl, err := dagger.Client(ctx)
			if err != nil {
				return nil, err
			}

			localDirs := make(map[string]dagger.FSID)
			for localID := range solveOpts.LocalDirs {
				res := struct {
					Core struct {
						ClientDir struct {
							Id dagger.FSID
						}
					}
				}{}
				resp := &graphql.Response{Data: &res}
				err = cl.MakeRequest(ctx,
					&graphql.Request{
						Query: `
							query ClientDir($id: String!) {
								core {
									clientdir(id: $id) {
										id
									}
								}
							}`,
						Variables: map[string]any{
							"id": localID,
						},
					},
					resp,
				)
				if err != nil {
					return nil, err
				}
				if len(resp.Errors) > 0 {
					return nil, resp.Errors
				}
				localDirs[localID] = res.Core.ClientDir.Id
			}

			if fn == nil {
				return nil, nil
			}

			outputFs, err := fn(ctx, localDirs, secretIDToKey)
			if err != nil {
				return nil, err
			}

			var result *bkgw.Result
			if outputFs != "" {
				pbdef, err := (&api.Filesystem{ID: api.FSID(outputFs)}).ToDefinition()
				if err != nil {
					return nil, err
				}
				res, err := gw.Solve(ctx, bkgw.SolveRequest{Evaluate: true, Definition: pbdef})
				if err != nil {
					return nil, err
				}
				result = res
			}
			if result == nil {
				result = bkgw.NewResult()
			}

			if startOpts.DevServer != 0 {
				http.Handle("/", router)
				return nil, http.ListenAndServe(fmt.Sprintf(":%d", startOpts.DevServer), nil)
			}

			return result, nil
		}, ch)
		return err
	})
	eg.Go(func() error {
		warn, err := progressui.DisplaySolveStatus(context.TODO(), "", nil, os.Stderr, ch)
		for _, w := range warn {
			fmt.Fprintf(os.Stderr, "=> %s\n", w.Short)
		}
		return err
	})
	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}

func withInMemoryAPIClient(ctx context.Context, router *router.Router) context.Context {
	return dagger.WithHTTPClient(ctx, &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				// TODO: not efficient, but whatever
				serverConn, clientConn := net.Pipe()

				go func() {
					l := &singleConnListener{conn: serverConn}

					srv := &http.Server{
						Handler: router,
					}
					srv.Serve(l)
				}()

				return clientConn, nil
			},
		},
	})
}

// converts a pre-existing net.Conn into a net.Listener that returns the conn
type singleConnListener struct {
	conn net.Conn
}

func (l *singleConnListener) Accept() (net.Conn, error) {
	if l.conn == nil {
		return nil, io.ErrClosedPipe
	}
	c := l.conn
	l.conn = nil
	return c, nil
}

func (l *singleConnListener) Addr() net.Addr {
	return nil
}

func (l *singleConnListener) Close() error {
	return nil
}

func detectPlatform(ctx context.Context, c *bkclient.Client) (*specs.Platform, error) {
	w, err := c.ListWorkers(ctx)
	if err != nil {
		return nil, fmt.Errorf("error detecting platform %w", err)
	}

	if len(w) > 0 && len(w[0].Platforms) > 0 {
		dPlatform := w[0].Platforms[0]
		return &dPlatform, nil
	}
	defaultPlatform := platforms.DefaultSpec()
	return &defaultPlatform, nil
}

func Shell(ctx context.Context, gw bkgw.Client, platform specs.Platform, inputFS dagger.FSID) error {
	baseDef, err := llb.Image("alpine:3.15").Marshal(ctx, llb.Platform(platform))
	if err != nil {
		return err
	}
	baseRes, err := gw.Solve(ctx, bkgw.SolveRequest{
		Definition: baseDef.ToPB(),
	})
	if err != nil {
		return err
	}
	baseRef, err := baseRes.SingleRef()
	if err != nil {
		return err
	}

	pbdef, err := (&api.Filesystem{ID: api.FSID(inputFS)}).ToDefinition()
	if err != nil {
		return err
	}
	fsRes, err := gw.Solve(ctx, bkgw.SolveRequest{
		Definition: pbdef,
	})
	if err != nil {
		return err
	}
	fsRef, err := fsRes.SingleRef()
	if err != nil {
		return err
	}

	ctr, err := gw.NewContainer(ctx, bkgw.NewContainerRequest{
		Mounts: []bkgw.Mount{
			{
				Dest:      "/",
				Ref:       baseRef,
				MountType: pb.MountType_BIND,
			},
			{
				Dest:      "/output",
				Ref:       fsRef,
				MountType: pb.MountType_BIND,
			},
		},
	})
	if err != nil {
		return err
	}
	proc, err := ctr.Start(ctx, bkgw.StartRequest{
		Args:   []string{"/bin/sh"},
		Cwd:    "/output",
		Tty:    true,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
	if err != nil {
		return err
	}
	termState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer terminal.Restore(int(os.Stdin.Fd()), termState)
	return proc.Wait()
}
