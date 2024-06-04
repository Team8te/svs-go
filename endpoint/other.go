package endpoint

import (
	"fmt"
	"net/http"

	"github.com/Team8te/svs-go/protocol/rtmp/rtmprelay"
	log "github.com/sirupsen/logrus"
)

// http://127.0.0.1:8090/control/pull?&oper=start&app=live&name=123456&url=rtmp://192.168.16.136/live/123456
func (s *Endpoint) handlePull(w http.ResponseWriter, req *http.Request) {
	var retString string
	var err error

	res := &Response{
		w:      w,
		Data:   nil,
		Status: 200,
	}

	defer res.SendJson()

	if req.ParseForm() != nil {
		res.Status = 400
		res.Data = "url: /control/pull?&oper=start&app=live&name=123456&url=rtmp://192.168.16.136/live/123456"
		return
	}

	oper := req.Form.Get("oper")
	app := req.Form.Get("app")
	name := req.Form.Get("name")
	url := req.Form.Get("url")

	log.Debugf("control pull: oper=%v, app=%v, name=%v, url=%v", oper, app, name, url)
	if (len(app) <= 0) || (len(name) <= 0) || (len(url) <= 0) {
		res.Status = 400
		res.Data = "control push parameter error, please check them."
		return
	}

	remoteurl := "rtmp://127.0.0.1" + s.rtmpAddr + "/" + app + "/" + name
	localurl := url

	keyString := "pull:" + app + "/" + name
	if oper == "stop" {
		pullRtmprelay, found := s.session[keyString]

		if !found {
			retString = fmt.Sprintf("session key[%s] not exist, please check it again.", keyString)
			res.Status = 400
			res.Data = retString
			return
		}
		log.Debugf("rtmprelay stop push %s from %s", remoteurl, localurl)
		pullRtmprelay.Stop()

		delete(s.session, keyString)
		retString = fmt.Sprintf("<h1>push url stop %s ok</h1></br>", url)
		res.Status = 400
		res.Data = retString
		log.Debugf("pull stop return %s", retString)
	} else {
		pullRtmprelay := rtmprelay.NewRtmpRelay(&localurl, &remoteurl)
		log.Debugf("rtmprelay start push %s from %s", remoteurl, localurl)
		err = pullRtmprelay.Start()
		if err != nil {
			res.Status = 400
			retString = fmt.Sprintf("push error=%v", err)
		} else {
			s.session[keyString] = pullRtmprelay
			retString = fmt.Sprintf("<h1>pull url start %s ok</h1></br>", url)
		}

		res.Data = retString
		log.Debugf("pull start return %s", retString)
	}
}

// http://127.0.0.1:8090/control/push?&oper=start&app=live&name=123456&url=rtmp://192.168.16.136/live/123456
func (s *Endpoint) handlePush(w http.ResponseWriter, req *http.Request) {
	var retString string
	var err error

	res := &Response{
		w:      w,
		Data:   nil,
		Status: 200,
	}

	defer res.SendJson()

	if req.ParseForm() != nil {
		res.Data = "url: /control/push?&oper=start&app=live&name=123456&url=rtmp://192.168.16.136/live/123456"
		return
	}

	oper := req.Form.Get("oper")
	app := req.Form.Get("app")
	name := req.Form.Get("name")
	url := req.Form.Get("url")

	log.Debugf("control push: oper=%v, app=%v, name=%v, url=%v", oper, app, name, url)
	if (len(app) <= 0) || (len(name) <= 0) || (len(url) <= 0) {
		res.Data = "control push parameter error, please check them."
		return
	}

	localurl := "rtmp://127.0.0.1" + s.rtmpAddr + "/" + app + "/" + name
	remoteurl := url

	keyString := "push:" + app + "/" + name
	if oper == "stop" {
		pushRtmprelay, found := s.session[keyString]
		if !found {
			retString = fmt.Sprintf("<h1>session key[%s] not exist, please check it again.</h1>", keyString)
			res.Data = retString
			return
		}
		log.Debugf("rtmprelay stop push %s from %s", remoteurl, localurl)
		pushRtmprelay.Stop()

		delete(s.session, keyString)
		retString = fmt.Sprintf("<h1>push url stop %s ok</h1></br>", url)
		res.Data = retString
		log.Debugf("push stop return %s", retString)
	} else {
		pushRtmprelay := rtmprelay.NewRtmpRelay(&localurl, &remoteurl)
		log.Debugf("rtmprelay start push %s from %s", remoteurl, localurl)
		err = pushRtmprelay.Start()
		if err != nil {
			retString = fmt.Sprintf("push error=%v", err)
		} else {
			retString = fmt.Sprintf("<h1>push url start %s ok</h1></br>", url)
			s.session[keyString] = pushRtmprelay
		}

		res.Data = retString
		log.Debugf("push start return %s", retString)
	}
}
