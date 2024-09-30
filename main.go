package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"path"
	"runtime"
	"time"

	"github.com/Team8te/svs-go/configure"
	"github.com/Team8te/svs-go/endpoint"
	"github.com/Team8te/svs-go/protocol/center"
	"github.com/Team8te/svs-go/protocol/hls"
	"github.com/Team8te/svs-go/protocol/httpflv"
	"github.com/Team8te/svs-go/protocol/rtmp"
	"github.com/Team8te/svs-go/repo"

	log "github.com/sirupsen/logrus"
)

var VERSION = "master"

func makeHls() *hls.HLSServer {
	hlsAddr := configure.Config.GetString("hls_addr")
	hlsListen, err := net.Listen("tcp", hlsAddr)
	if err != nil {
		log.Fatal(err)
	}

	hlsServer := hls.NewHLSServer(hlsListen)
	return hlsServer
}

func makeRtmp(mediaCenter *center.MediaCenter) *rtmp.Server {
	rtmpAddr := configure.Config.GetString("rtmp_addr")
	isRtmps := configure.Config.GetBool("enable_rtmps")

	var rtmpListen net.Listener
	if isRtmps {
		certPath := configure.Config.GetString("rtmps_cert")
		keyPath := configure.Config.GetString("rtmps_key")
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			log.Fatal(err)
		}

		rtmpListen, err = tls.Listen("tcp", rtmpAddr, &tls.Config{
			Certificates: []tls.Certificate{cert},
		})
		if err != nil {
			log.Fatal(err)
		}
	} else {
		var err error
		rtmpListen, err = net.Listen("tcp", rtmpAddr)
		if err != nil {
			log.Fatal(err)
		}
	}
	defer func() {
		if r := recover(); r != nil {
			log.Error("RTMP server panic: ", r)
		}
	}()
	rtmpServer := rtmp.NewServer(rtmpListen, rtmp.MakeMediaCenter(mediaCenter))
	return rtmpServer
}

func startHTTPFlv() {
	httpflvAddr := configure.Config.GetString("httpflv_addr")

	flvListen, err := net.Listen("tcp", httpflvAddr)
	if err != nil {
		log.Fatal(err)
	}

	hdlServer := httpflv.NewServer(nil)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error("HTTP-FLV server panic: ", r)
			}
		}()
		log.Info("HTTP-FLV listen On ", httpflvAddr)
		hdlServer.Serve(flvListen)
	}()
}

func startAPI(r *repo.Repo) {
	apiAddr := configure.Config.GetString("api_addr")
	rtmpAddr := configure.Config.GetString("rtmp_addr")

	if apiAddr != "" {
		opListen, err := net.Listen("tcp", apiAddr)
		if err != nil {
			log.Fatal(err)
		}
		opServer := endpoint.NewEndpoint(rtmpAddr, r)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error("HTTP-API server panic: ", r)
				}
			}()
			log.Info("HTTP-API listen On ", apiAddr)
			opServer.Serve(opListen)
		}()
	}
}

func init() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := path.Base(f.File)
			return fmt.Sprintf("%s()", f.Function), fmt.Sprintf(" %s:%d", filename, f.Line)
		},
	})
	log.SetLevel(log.DebugLevel)
}

type app interface {
	Run(ctx context.Context)
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Error("livego panic: ", r)
			time.Sleep(1 * time.Second)
		}
	}()

	log.Infof(`
     _     _            ____
    | |   (_)_   _____ / ___| ___
    | |   | \ \ / / _ \ |  _ / _ \
    | |___| |\ V /  __/ |_| | (_) |
    |_____|_| \_/ \___|\____|\___/
        version: %s
	`, VERSION)

	capps := configure.Applications{}
	r := repo.NewRepo()
	configure.Config.UnmarshalKey("server", &capps)
	apps := make([]app, 0)
	hls := makeHls()
	mediaCenter := center.MakeMediaCenter(hls)
	apps = append(apps, makeRtmp(mediaCenter))
	for _, app := range capps {
		if app.Hls {
			apps = append(apps, hls)
		}
		if app.Flv {
			startHTTPFlv()
		}
		if app.Api {
			startAPI(r)
		}
	}

	ctx := context.TODO()

	for _, a := range apps {
		app := a
		go app.Run(ctx)
	}

	select {}
}
