package endpoint

import (
	"context"
	"encoding/json"
	"net"
	"net/http"

	"github.com/Team8te/svs-go/configure"
	"github.com/Team8te/svs-go/ds"
	"github.com/Team8te/svs-go/pkg/av"
	"github.com/Team8te/svs-go/protocol/rtmp/rtmprelay"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	"github.com/dgrijalva/jwt-go"
	log "github.com/sirupsen/logrus"
)

type Response struct {
	w      http.ResponseWriter
	Status int         `json:"status"`
	Data   interface{} `json:"data"`
}

func (r *Response) SendJson() (int, error) {
	resp, _ := json.Marshal(r)
	r.w.Header().Set("Content-Type", "application/json")
	r.w.WriteHeader(r.Status)
	return r.w.Write(resp)
}

type roomService interface {
	CreateRoom(ctx context.Context, name string) (*ds.Room, error)
}

type Endpoint struct {
	handler  av.Handler
	session  map[string]*rtmprelay.RtmpRelay
	rtmpAddr string
	rs       roomService
}

func NewEndpoint(h av.Handler, rtmpAddr string, rs roomService) *Endpoint {
	return &Endpoint{
		handler:  h,
		session:  make(map[string]*rtmprelay.RtmpRelay),
		rtmpAddr: rtmpAddr,
		rs:       rs,
	}
}

func JWTMiddleware(next http.Handler) http.Handler {
	isJWT := len(configure.Config.GetString("jwt.secret")) > 0
	if !isJWT {
		return next
	}

	log.Info("Using JWT middleware")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var algorithm jwt.SigningMethod
		if len(configure.Config.GetString("jwt.algorithm")) > 0 {
			algorithm = jwt.GetSigningMethod(configure.Config.GetString("jwt.algorithm"))
		}

		if algorithm == nil {
			algorithm = jwt.SigningMethodHS256
		}

		jwtMiddleware := jwtmiddleware.New(jwtmiddleware.Options{
			Extractor: jwtmiddleware.FromFirst(jwtmiddleware.FromAuthHeader, jwtmiddleware.FromParameter("jwt")),
			ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
				return []byte(configure.Config.GetString("jwt.secret")), nil
			},
			SigningMethod: algorithm,
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err string) {
				res := &Response{
					w:      w,
					Status: 403,
					Data:   err,
				}
				res.SendJson()
			},
		})

		jwtMiddleware.HandlerWithNext(w, r, next.ServeHTTP)
	})
}

func (s *Endpoint) Serve(l net.Listener) error {
	mux := http.NewServeMux()

	mux.Handle("/statics/", http.StripPrefix("/statics/", http.FileServer(http.Dir("statics"))))

	mux.HandleFunc("/control/push", func(w http.ResponseWriter, r *http.Request) {
		s.handlePush(w, r)
	})
	mux.HandleFunc("/control/pull", func(w http.ResponseWriter, r *http.Request) {
		s.handlePull(w, r)
	})
	mux.HandleFunc("/room/create", func(w http.ResponseWriter, r *http.Request) {
		resp := s.createRoomHandler(w, r)
		resp.SendJson()
	})
	mux.HandleFunc("/stat/livestat", func(w http.ResponseWriter, r *http.Request) {
		resp := s.getLiveStaticsHandler(w, r)
		resp.SendJson()
	})
	http.Serve(l, JWTMiddleware(mux))
	return nil
}
